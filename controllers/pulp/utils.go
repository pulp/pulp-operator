package pulp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crypt_rand "crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	caConfigMapName = "user-ca-bundle"
)

// Generate a random string with length pwdSize
func createPwd(pwdSize int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pwd := make([]byte, pwdSize)
	for i := range pwd {
		pwd[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwd)
}

// Retrieve specific keys from secret object
func (r *PulpReconciler) retrieveSecretData(ctx context.Context, secretName, secretNamespace string, required bool, keys ...string) (map[string]string, error) {
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, found)
	if err != nil {
		return nil, err
	}

	secret := map[string]string{}
	for _, key := range keys {
		// all provided keys should be present on secret, if not return error
		if required && found.Data[key] == nil {
			return nil, fmt.Errorf("could not find %v key in %v secret", key, secretName)
		}

		// if the keys provided are not mandatory and are also not defined, just skip them
		if !required && found.Data[key] == nil {
			continue
		}
		secret[key] = string(found.Data[key])
	}

	return secret, nil
}

// Get signing key fingerprint from secret object
func (r *PulpReconciler) getSigningKeyFingerprint(secretName, secretNamespace string) (string, error) {

	ctx := context.TODO()
	secretData, err := r.retrieveSecretData(ctx, secretName, secretNamespace, true, "signing_service.gpg")
	if err != nil {
		return "", err
	}

	// "convert" to Reader to be used by ReadArmoredKeyRing
	secretReader := strings.NewReader(secretData["signing_service.gpg"])

	// Read public key
	keyring, err := openpgp.ReadArmoredKeyRing(secretReader)
	if err != nil {
		return "", errors.New("Read Key Ring Error! " + err.Error())
	}

	fingerPrint := keyring[0].PrimaryKey.Fingerprint
	return strings.ToUpper(hex.EncodeToString(fingerPrint[:])), nil

}

func getPulpSetting(pulp *repomanagerv1alpha1.Pulp, key string) string {
	settings := pulp.Spec.PulpSettings.Raw
	var settingsJson map[string]interface{}
	json.Unmarshal(settings, &settingsJson)

	v := settingsJson[key]
	// default values
	if v == nil {
		switch key {
		case "api_root":
			return "/pulp/"
		case "content_path_prefix":
			return "/pulp/content/"
		case "galaxy_collection_signing_service":
			return "ansible-default"
		case "galaxy_container_signing_service":
			return "container-default"
		}
	}
	switch v.(type) {
	case map[string]interface{}:
		rawMapping, _ := json.Marshal(v)
		return fmt.Sprintln(strings.Replace(string(rawMapping), "\"", "'", -1))
	default:
		return fmt.Sprintf("%v", v)
	}

}

// addCustomPulpSettings appends custom settings defined in Pulp CR to settings.py
func addCustomPulpSettings(pulp *repomanagerv1alpha1.Pulp, current_settings string) string {
	settings := pulp.Spec.PulpSettings.Raw
	var settingsJson map[string]interface{}
	json.Unmarshal(settings, &settingsJson)

	var convertedSettings string
	for k, v := range settingsJson {
		if strings.Contains(current_settings, strings.ToUpper(k)) {
			lines := strings.Split(current_settings, strings.ToUpper(k))
			current_settings = lines[0] + strings.Join(strings.Split(lines[1], "\n")[1:], "\n")
		}
		switch v.(type) {
		case map[string]interface{}:
			rawMapping, _ := json.Marshal(v)
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", strings.Replace(string(rawMapping), "\"", "'", -1))
		default:
			convertedSettings = convertedSettings + fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(k), v)
		}
	}

	return current_settings + convertedSettings
}

func genTokenAuthKey() (string, string) {
	newKey, _ := ecdsa.GenerateKey(elliptic.P256(), crypt_rand.Reader)
	pubKeyDER, _ := x509.MarshalPKIXPublicKey(&newKey.PublicKey)
	ecDER, _ := x509.MarshalECPrivateKey(newKey)

	privateKey := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}))
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER}))

	return privateKey, publicKey
}

func (r *PulpReconciler) updateStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, conditionStatus metav1.ConditionStatus, conditionType, conditionReason, conditionMessage string) {

	// if we are updating a status it means that operator didn't finish its execution
	if v1.IsStatusConditionPresentAndEqual(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution", metav1.ConditionTrue) {
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionFalse,
			Reason:             "OperatorRunning",
			LastTransitionTime: metav1.Now(),
			Message:            pulp.Name + " operator tasks running",
		})
	}

	v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             conditionReason,
		LastTransitionTime: metav1.Now(),
		Message:            conditionMessage,
	})
	r.Status().Update(ctx, pulp)
}

// createEmptyConfigMap creates an empty ConfigMap that is used by CNO (Cluster Network Operator) to
// inject custom CA into containers
func (r *PulpReconciler) createEmptyConfigMap(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: caConfigMapName, Namespace: pulp.Namespace}, configMap)

	expected_cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caConfigMapName,
			Namespace: pulp.Namespace,
			Labels: map[string]string{
				"config.openshift.io/inject-trusted-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}

	// create the configmap if not found
	if err != nil && k8s_errors.IsNotFound(err) {
		log.V(1).Info("Creating a new empty ConfigMap")
		ctrl.SetControllerReference(pulp, expected_cm, r.Scheme)
		err = r.Create(ctx, expected_cm)
		if err != nil {
			log.Error(err, "Failed to create empty ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get empty ConfigMap")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
func mountCASpec(pulp *repomanagerv1alpha1.Pulp, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]corev1.Volume, []corev1.VolumeMount) {

	if pulp.Spec.TrustedCa {

		// trustedCAVolume contains the configmap with the custom ca bundle
		trustedCAVolume := corev1.Volume{
			Name: "trusted-ca",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: caConfigMapName,
					},
					Items: []corev1.KeyToPath{
						{Key: "ca-bundle.crt", Path: "tls-ca-bundle.pem"},
					},
				},
			},
		}
		volumes = append(volumes, trustedCAVolume)

		// trustedCAMount defines the mount point of the configmap
		// with the custom ca bundle
		trustedCAMount := corev1.VolumeMount{
			Name:      "trusted-ca",
			MountPath: "/etc/pki/ca-trust/extracted/pem",
			ReadOnly:  true,
		}
		volumeMounts = append(volumeMounts, trustedCAMount)
	}

	return volumes, volumeMounts
}

// checkImmutableFields verifies if a user tried to modify an immutable field and rollback
// the change if so
func (r *PulpReconciler) checkImmutableFields(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, field string, log logr.Logger) bool {

	// access the field by its string name
	// for fieldSpec we need to pass it as a reference because we will need to change
	// its value back in case of immutable field
	fieldSpec := reflect.Indirect(reflect.ValueOf(&pulp.Spec)).FieldByName(field)

	// for fieldStatus, as we just need its content, we dont need to get a
	// pointer to it
	fieldStatus := reflect.ValueOf(pulp.Status).FieldByName(field)

	// first we need to call the Interface() method to use the field as interface{}
	// then we assert that the interface{} is a string so we can check if the len > 0
	// if len>0 means that the field was previously defined
	// after that we check if the content from .status.field is different from spec.field
	// if so we should rollback the modification
	if len(fieldStatus.Interface().(string)) > 0 && fieldStatus.Interface() != fieldSpec.Interface() {
		patch := client.MergeFrom(pulp.DeepCopy())

		// set pulp.spec.<field> back with the value from .status
		fieldSpec.SetString(fieldStatus.Interface().(string))

		// we are using patch here because we need to modify only a specific field.
		// if we had used update it would fill a lot of other fields with default values
		// which would also trigger a reconciliation loop
		r.Patch(ctx, pulp, patch)
		err := fmt.Errorf("%s field is immutable", field)
		log.Error(err, "Could not update "+field+" field")
		return true
	}
	return false
}

// updateIngressType will check the current definition of ingress_type and will handle the different
// modification scenarios
func (r *PulpReconciler) updateIngressType(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) {

	// if pulp CR was defined with route and user modified it to anything else
	// delete all routes with operator's labels
	// remove route .status.conditions
	// update .status.ingress_type = nodeport
	if strings.ToLower(pulp.Status.IngressType) == "route" && strings.ToLower(pulp.Spec.IngressType) != "route" {

		route := &routev1.Route{}
		routesLabelSelector := map[string]string{
			"pulp_cr": pulp.Name,
			"owner":   "pulp-dev",
		}
		listOpts := []client.DeleteAllOfOption{
			client.InNamespace(pulp.Namespace),
			client.MatchingLabels(routesLabelSelector),
		}
		r.DeleteAllOf(ctx, route, listOpts...)
		routeConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Route-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, routeConditionType)

		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}

	// if pulp CR was defined with nodeport and user modified it to anything else
	// delete all pulp-web resources
	// remove pulp-web .status.conditions
	// update .status.ingress_type = route
	// we will not remove configmap to avoid losing resources that are potentially unrecoverable
	if strings.ToLower(pulp.Status.IngressType) == "nodeport" && strings.ToLower(pulp.Spec.IngressType) != "nodeport" {
		webDeployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment); err != nil {
			return
		}
		r.Delete(ctx, webDeployment)

		webSvc := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web-svc", Namespace: pulp.Namespace}, webSvc); err != nil {
			return
		}
		r.Delete(ctx, webSvc)

		webConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, webConditionType)

		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}

}

// [TODO] Find a generic way to assert the object type to avoid these repetitive blocks of code
// reconcileObject will check if the definition from Pulp CR is reflecting the current
// object state and if not will synchronize the configuration
func (r *PulpReconciler) reconcileObject(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, expectedState, currentState interface{}, conditionType string, log logr.Logger) error {

	switch expectedState := expectedState.(type) {
	case *routev1.Route:
		objKind := "route"
		objName := expectedState.Name
		currentState := currentState.(*routev1.Route)
		// Ensure objects are as expected
		if !equality.Semantic.DeepDerivative(expectedState.Spec, currentState.Spec) {
			log.Info("The " + objKind + " " + objName + " has been modified! Reconciling ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)
			expectedState.SetResourceVersion(currentState.GetResourceVersion())
			if err := r.Update(ctx, expectedState); err != nil {
				log.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
				return err
			}
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Reconciled "+objName+" "+objKind)
		}
	case *corev1.Service:
		objKind := "service"
		objName := expectedState.Name
		currentState := currentState.(*corev1.Service)

		// if NodePort field is REMOVED we dont need to do anything
		// kubernetes will define a new nodeport automatically
		if pulp.Spec.NodePort == 0 {
			return nil
		}

		// Ensure objects are as expected
		if !equality.Semantic.DeepDerivative(expectedState.Spec, currentState.Spec) {
			log.Info("The " + objKind + " " + objName + " has been modified! Reconciling ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)
			if err := r.Update(ctx, expectedState); err != nil {
				log.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
				return err
			}
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Reconciled "+objName+" "+objKind)
		}
	}

	return nil
}

// reconcileMetadata is a method to handle only .metadata.{labels,annotations} reconciliation
// for some reason, if we try to use DeepDerivative like
//    if !equality.Semantic.DeepDerivative(expectedState.(*routev1.Route), currentState.(*routev1.Route)) ...
// it will get into an infinite loop
func (r *PulpReconciler) reconcileMetadata(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, expectedState, currentState interface{}, conditionType string, log logr.Logger) error {

	switch expectedState.(type) {
	case *routev1.Route:
		objKind := "route"
		objName := expectedState.(*routev1.Route).Name

		// metadata fields
		metadataFields := []string{"Labels", "Annotations"}

		for _, field := range metadataFields {
			currentField := reflect.ValueOf(currentState.(*routev1.Route).ObjectMeta).FieldByName(field).Interface()
			expectedField := reflect.ValueOf(expectedState.(*routev1.Route).ObjectMeta).FieldByName(field).Interface()
			if !equality.Semantic.DeepDerivative(currentField, expectedField) ||
				!equality.Semantic.DeepDerivative(expectedField, currentField) {

				log.Info("The " + objKind + " " + objName + " has been modified! Reconciling ...")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)
				expectedState.(*routev1.Route).SetResourceVersion(currentState.(*routev1.Route).GetResourceVersion())
				if err := r.Update(ctx, expectedState.(*routev1.Route)); err != nil {
					log.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
					r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
					return err
				}
				r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Reconciled "+objName+" "+objKind)
			}
		}
	}

	return nil
}
