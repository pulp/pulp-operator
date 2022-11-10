package repo_manager

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crypt_rand "crypto/rand"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
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

type immutableField struct {
	FieldName string
	FieldPath interface{}
}

// statusReturn is used to control goroutines execution
type statusReturn struct {
	ctrl.Result
	error
}

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
func (r *RepoManagerReconciler) retrieveSecretData(ctx context.Context, secretName, secretNamespace string, required bool, keys ...string) (map[string]string, error) {
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, found)
	if err != nil {
		return nil, err
	}

	secret := map[string]string{}
	for _, key := range keys {
		// all provided keys should be present on secret, if not return error
		if required && found.Data[key] == nil {
			return nil, fmt.Errorf("could not find \"%v\" key in %v secret", key, secretName)
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
func (r *RepoManagerReconciler) getSigningKeyFingerprint(secretName, secretNamespace string) (string, error) {

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

// updateStatus will set the new condition value for a .status.conditions[]
// it will also set Pulp-Operator-Finished-Execution to false
func (r *RepoManagerReconciler) updateStatus(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, conditionStatus metav1.ConditionStatus, conditionType, conditionReason, conditionMessage string) {

	// if we are updating a status it means that operator didn't finish its execution
	// set Pulp-Operator-Finished-Execution to false
	if v1.IsStatusConditionPresentAndEqual(pulp.Status.Conditions, cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType)+"-Operator-Finished-Execution", metav1.ConditionTrue) {
		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Operator-Finished-Execution",
			Status:             metav1.ConditionFalse,
			Reason:             "OperatorRunning",
			LastTransitionTime: metav1.Now(),
			Message:            pulp.Name + " operator tasks running",
		})
	}

	// we will only update if the current condition is not as expected
	if !v1.IsStatusConditionPresentAndEqual(pulp.Status.Conditions, conditionType, conditionStatus) {

		v1.SetStatusCondition(&pulp.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             conditionStatus,
			Reason:             conditionReason,
			LastTransitionTime: metav1.Now(),
			Message:            conditionMessage,
		})

		r.Status().Update(ctx, pulp)
	}
}

// updatCRField patches fieldName in Pulp CR with fieldValue
func (r *RepoManagerReconciler) updateCRField(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, fieldName, fieldValue string) error {
	field := reflect.Indirect(reflect.ValueOf(&pulp.Spec)).FieldByName(fieldName)

	// we will only set the field (with default values) if there is nothing defined yet
	// this is to avoid overwriting user definition
	if len(field.Interface().(string)) == 0 {
		patch := client.MergeFrom(pulp.DeepCopy())
		field.SetString(fieldValue)
		if err := r.Patch(ctx, pulp, patch); err != nil {
			return err
		}
	}
	return nil
}

// createEmptyConfigMap creates an empty ConfigMap that is used by CNO (Cluster Network Operator) to
// inject custom CA into containers
func (r *RepoManagerReconciler) createEmptyConfigMap(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

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
func (r *RepoManagerReconciler) checkImmutableFields(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, field immutableField, log logr.Logger) bool {

	fieldSpec := reflect.Value{}

	// access the field by its string name
	// for fieldSpec we need to pass it as a reference because we will need to change
	// its value back in case of immutable field
	switch field.FieldPath.(type) {
	case repomanagerv1alpha1.PulpSpec:
		fieldSpec = reflect.Indirect(reflect.ValueOf(&pulp.Spec)).FieldByName(field.FieldName)
	case repomanagerv1alpha1.Cache:
		fieldSpec = reflect.Indirect(reflect.ValueOf(&pulp.Spec.Cache)).FieldByName(field.FieldName)
	}

	// for fieldStatus, as we just need its content, we dont need to get a
	// pointer to it
	fieldStatus := reflect.ValueOf(pulp.Status).FieldByName(field.FieldName)

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
		err := fmt.Errorf("%s field is immutable", field.FieldName)
		log.Error(err, "Could not update "+field.FieldName+" field")
		return true
	}
	return false
}

// updateIngressType will check the current definition of ingress_type and will handle the different
// modification scenarios
func (r *RepoManagerReconciler) updateIngressType(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) {

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

	// if pulp CR was defined with ingress and user modified it to anything else
	// delete all ingresss with operator's labels
	// remove ingress .status.conditions
	if strings.ToLower(pulp.Status.IngressType) == "ingress" && strings.ToLower(pulp.Spec.IngressType) != "ingress" {

		ingress := &netv1.Ingress{}
		ingresssLabelSelector := map[string]string{
			"pulp_cr": pulp.Name,
			"owner":   "pulp-dev",
		}
		listOpts := []client.DeleteAllOfOption{
			client.InNamespace(pulp.Namespace),
			client.MatchingLabels(ingresssLabelSelector),
		}
		r.DeleteAllOf(ctx, ingress, listOpts...)
		ingressConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Ingress-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, ingressConditionType)

		pulp.Status.IngressType = pulp.Spec.IngressType

		if len(pulp.Status.IngressClassName) > 0 {
			pulp.Status.IngressClassName = ""
		}

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

// updateIngressClass will check the current definition of ingress_class_name and will handle the different
// modification scenarios
func (r *RepoManagerReconciler) updateIngressClass(ctx context.Context, pulp *repomanagerv1alpha1.Pulp) {

	// if the new one uses nginx controller
	if r.isNginxIngress(pulp) {

		// remove pulp-web components
		webDeployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment); err == nil {
			r.Delete(ctx, webDeployment)
		}

		webSvc := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web-svc", Namespace: pulp.Namespace}, webSvc); err == nil {
			r.Delete(ctx, webSvc)
		}
		webConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, webConditionType)

		// or the new one does not use nginx controller anymore
	} else {
		// remove ingress resource
		ingress := &netv1.Ingress{}
		r.Get(ctx, types.NamespacedName{Name: pulp.Name, Namespace: pulp.Namespace}, ingress)
		r.Delete(ctx, ingress)

		ingressConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Ingress-Ready"
		v1.RemoveStatusCondition(&pulp.Status.Conditions, ingressConditionType)
	}

	pulp.Status.IngressClassName = pulp.Spec.IngressClassName
	r.Status().Update(ctx, pulp)
}

// reconcileObject will check if the definition from Pulp CR is reflecting the current
// object state and if not will synchronize the configuration
func reconcileObject(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string) (bool, error) {

	var objKind string

	// checkFunction is the function to check if resource is equal to expected
	checkFunction := checkSpecModification
	switch expectedState.(type) {
	case *routev1.Route:
		objKind = "Route"
	case *netv1.Ingress:
		objKind = "Ingress"
	case *corev1.Service:
		objKind = "Service"

		// if NodePort field is REMOVED we dont need to do anything
		// kubernetes will define a new nodeport automatically
		if funcResources.Pulp.Spec.NodePort == 0 {
			return false, nil
		}
	case *appsv1.Deployment:
		objKind = "Deployment"
		checkFunction = checkDeploymentSpec
	default:
		return false, nil
	}

	return updateObject(funcResources, checkFunction, objKind, conditionType, "Spec", expectedState, currentState)
}

// reconcileMetadata is a method to handle only .metadata.{labels,annotations} reconciliation
// for some reason, if we try to use DeepDerivative like
//    if !equality.Semantic.DeepDerivative(expectedState.(*routev1.Route), currentState.(*routev1.Route)) ...
// it will get into an infinite loop
func reconcileMetadata(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string) (bool, error) {

	objKind := ""
	switch expectedState.(type) {
	case *routev1.Route:
		objKind = "Route"
	case *netv1.Ingress:
		objKind = "Ingress"
	default:
		return false, nil
	}

	metadataFields := []string{"Labels", "Annotations"}
	for _, field := range metadataFields {
		if requeue, err := updateObject(funcResources, checkMetadataModification, objKind, conditionType, field, expectedState, currentState); err != nil || requeue {
			return requeue, err
		}
	}

	return false, nil
}

// checkDeployment returns true if a spec from deployment is not
// with the expected contents defined in Pulp CR
func checkDeploymentSpec(expectedState, currentState interface{}) bool {

	expectedSpec := expectedState.(appsv1.DeploymentSpec)
	currentSpec := currentState.(appsv1.DeploymentSpec)

	// Ensure the deployment template spec is as expected
	// https://github.com/kubernetes-sigs/kubebuilder/issues/592
	// * we are checking the []VolumeMounts because DeepDerivative will only make sure that
	//   what is in the expected definition is found in the current running deployment, which can have a gap
	//   in case of TrustedCA being true and eventually modified to false (the trusted-ca cm will not get unmounted).
	// * we are checking the .[]Containers.[]Volumemounts instead of []Volumes because reflect.DeepEqual(dep.Volumes,found.Volumes)
	//   identifies VolumeSource.EmptyDir being diff (not sure why).
	// * for NodeSelector, Tolerations, TopologySpreadConstraints, ResourceRequirements
	//     we are checking through Semantic.DeepEqual(expectedState.NodeSelector,currentState.NodeSelector) because the
	//     DeepDerivative(expectedState.Spec, currentState.Spec) only checks if {labels,tolerations,constraints} defined in expectedState are in currentState, but not
	//     if what is in the currentState is also in expectedState and we are not using reflect.DeepEqual because it will consider [] != nil
	return !equality.Semantic.DeepDerivative(expectedSpec, currentSpec) ||
		!reflect.DeepEqual(expectedSpec.Template.Spec.Containers[0].VolumeMounts, currentSpec.Template.Spec.Containers[0].VolumeMounts) ||
		!equality.Semantic.DeepEqual(expectedSpec.Template.Spec.NodeSelector, currentSpec.Template.Spec.NodeSelector) ||
		!equality.Semantic.DeepEqual(expectedSpec.Template.Spec.Tolerations, currentSpec.Template.Spec.Tolerations) ||
		!equality.Semantic.DeepEqual(expectedSpec.Template.Spec.TopologySpreadConstraints, currentSpec.Template.Spec.TopologySpreadConstraints) ||
		!equality.Semantic.DeepEqual(expectedSpec.Template.Spec.Containers[0].Resources, currentSpec.Template.Spec.Containers[0].Resources) ||
		!equality.Semantic.DeepEqual(expectedSpec.Template.Spec.Affinity, currentSpec.Template.Spec.Affinity)
}

// checkMetadataModification returns true if annotations or labels fields are not equal
// it is used to compare .metadata.annotations and .metadata.labels fields
// since these are map[string]string types
// we are using equality.Semantic.DeepEqual for both cases
//   (currentField, expectedField) and (expectedField, currentField)
// to evaluate map[string]nil == map[string]""(empty string) and
// map[string]"" == map[string]nil
func checkMetadataModification(currentField, expectedField interface{}) bool {
	return !equality.Semantic.DeepEqual(currentField, expectedField) || !equality.Semantic.DeepEqual(expectedField, currentField)
}

// checkSpecModification returns true if .spec fields present on A are equal to
// what is in B
func checkSpecModification(currentField, expectedField interface{}) bool {
	return !equality.Semantic.DeepDerivative(currentField, expectedField)
}

// updateObject is a function that verifies if an object has been modified
// and update, if necessary, with the expected config
func updateObject(funcResources FunctionResources, modified func(interface{}, interface{}) bool, objKind, conditionType, field string, expectedState, currentState client.Object) (bool, error) {

	// get object name
	objName := reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName("Name").Interface().(string)
	var currentField, expectedField interface{}

	// Get the interface value based on "field" parameter
	if field == "Annotations" || field == "Labels" {
		currentField = reflect.Indirect(reflect.ValueOf(currentState)).FieldByName("ObjectMeta").FieldByName(field).Interface()
		expectedField = reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName("ObjectMeta").FieldByName(field).Interface()
	} else if field == "Spec" {
		currentField = reflect.Indirect(reflect.ValueOf(currentState)).FieldByName(field).Interface()
		expectedField = reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName(field).Interface()
	}

	if modified(expectedField, currentField) {
		funcResources.Logger.Info("The " + objKind + " " + objName + " has been modified! Reconciling ...")
		funcResources.RepoManagerReconciler.updateStatus(funcResources.Context, funcResources.Pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)

		// for ingress and/or routes we need to "manually" update ResourceVersion fields
		// to avoid the issue
		//    "the object has been modified; please apply your changes to the latest version and try again"
		if isRouteOrIngress(expectedState) {
			reflect.ValueOf(expectedState).MethodByName("SetResourceVersion").Call(reflect.ValueOf(currentState).MethodByName("GetResourceVersion").Call([]reflect.Value{}))
		}
		funcResources.RepoManagerReconciler.recorder.Event(funcResources.Pulp, corev1.EventTypeNormal, "Updating", "Reconciling "+objName+" "+objKind)
		if err := funcResources.RepoManagerReconciler.Update(funcResources.Context, expectedState); err != nil {
			funcResources.Logger.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
			funcResources.RepoManagerReconciler.updateStatus(funcResources.Context, funcResources.Pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
			return false, err
		}
		funcResources.RepoManagerReconciler.recorder.Event(funcResources.Pulp, corev1.EventTypeNormal, "Updated", "Reconciled "+objName+" "+objKind)
		return true, nil
	}
	return false, nil
}

// isRouteOrIngress is used to assert if the provided resource is an ingress or a route
func isRouteOrIngress(resource interface{}) bool {
	_, route := resource.(*routev1.Route)
	_, ingress := resource.(*netv1.Ingress)
	return route || ingress
}

// ResourceDefinition has the attributes of a Pulp Resource
type ResourceDefinition struct {
	// A Context carries a deadline, a cancellation signal, and other values across
	// API boundaries.
	context.Context
	// Type is used to define what Kubernetes resource should be provisioned
	Type interface{}
	// Name sets the resource name
	Name string
	// Alias is used in .status.conditions field
	Alias string
	// ConditionType is used to update .status.conditions with the current resource state
	ConditionType string
	// Pulp is the Schema for the pulps API
	*repomanagerv1alpha1.Pulp
}

// FunctionResources contains the list of arguments passed to create new Pulp resources
type FunctionResources struct {
	context.Context
	*repomanagerv1alpha1.Pulp
	logr.Logger
	*RepoManagerReconciler
}

// createPulpResource executes a set of instructions to provision Pulp resources
func (r *RepoManagerReconciler) createPulpResource(resource ResourceDefinition, createFunc func(FunctionResources) client.Object) (bool, error) {
	log := r.RawLogger
	var object client.Object
	objKind := ""

	// assert resource type
	switch resourceType := resource.Type.(type) {
	case *corev1.Secret:
		object = resourceType
		objKind = "Secret"
	case *appsv1.Deployment:
		object = resourceType
		objKind = "Deployment"
	case *corev1.Service:
		object = resourceType
		objKind = "Service"
	case *corev1.PersistentVolumeClaim:
		object = resourceType
		objKind = "PVC"
	}

	// define the list of parameters to pass to "provisioner" function
	funcResources := FunctionResources{Context: resource.Context, Pulp: resource.Pulp, Logger: log, RepoManagerReconciler: r}

	// set of instructions to create a resource (the following are almost the same for most of Pulp resources)
	// - we check if the resource exists
	// - if not we update Pulp CR status, add a log message and create the resource
	//   - if the resource provisioning fails we update the status, add an error message and return the error
	//   - if the resource is provisioned we create an event and return
	// - for any other error (besides resource not found) we add an error log and return error
	currentResource := object
	err := r.Get(resource.Context, types.NamespacedName{Name: resource.Name, Namespace: resource.Pulp.Namespace}, currentResource)
	if err != nil && k8s_errors.IsNotFound(err) {
		expectedResource := createFunc(funcResources)
		r.updateStatus(resource.Context, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "Creating"+resource.Alias+objKind, "Creating "+resource.Name+" "+objKind)
		log.Info("Creating a new "+resource.Name+" "+objKind, "Namespace", resource.Pulp.Namespace, "Name", resource.Name)
		err = r.Create(resource.Context, expectedResource)

		// special condition for api deployments where we need to provide a warning message
		// in case no storage type is provided
		if resource.Name == resource.Pulp.Name+"-api" && objKind == "Deployment" {
			controllers.CheckEmptyDir(resource.Pulp, controllers.PulpResource)
		}

		if err != nil {
			log.Error(err, "Failed to create new "+resource.Name+" "+objKind)
			r.updateStatus(resource.Context, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "ErrorCreating"+resource.Alias+objKind, "Failed to create "+resource.Name+" "+objKind+": "+err.Error())
			r.recorder.Event(resource.Pulp, corev1.EventTypeWarning, "Failed", "Failed to create a new "+resource.Name+" "+objKind)
			return false, err
		}
		r.recorder.Event(resource.Pulp, corev1.EventTypeNormal, "Created", resource.Name+" "+objKind+" created")
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+resource.Name+" "+objKind)
		return false, err
	}

	return false, nil
}

// createFernetKey creates a random key that will be used in "database_fields.symmetric.key"
func createFernetKey() string {
	key := [32]byte{}
	io.ReadFull(crypt_rand.Reader, key[:])
	return b64.StdEncoding.EncodeToString(key[:])
}

// needsRequeue will return true if the controller should trigger a new reconcile loop
func needsRequeue(err error, pulpController ctrl.Result) bool {
	return err != nil || !reflect.DeepEqual(pulpController, ctrl.Result{})
}

// isNginxIngress will check if ingress_type is defined as "ingress"
func isIngress(pulp *repomanagerv1alpha1.Pulp) bool {
	return strings.ToLower(pulp.Spec.IngressType) == "ingress"
}

// isNginxIngress returns true if pulp is defined with ingress_type==ingress and the controller of the ingresclass provided is a nginx
func (r *RepoManagerReconciler) isNginxIngress(pulp *repomanagerv1alpha1.Pulp) bool {
	return isIngress(pulp) && controllers.IsNginxIngressSupported(r, pulp.Spec.IngressClassName)
}
