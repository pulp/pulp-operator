package repo_manager

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crypt_rand "crypto/rand"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	pulp_ocp "github.com/pulp/pulp-operator/controllers/ocp"
	"golang.org/x/exp/maps"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type immutableField struct {
	FieldName string
	FieldPath interface{}
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

// sortKeys will return an ordered slice of strings defined with the keys from a.
// It is used to make sure that custom settings from pulp-server secret
// will be built in the same order and avoiding issues when verifying if its
// content is as expected. For example, this will avoid the controller having
// to check if
//
//	  pulp_settings:
//		    allowed_export_paths:
//		    - /tmp
//		    telemetry: falsew
//
// will converge into a secret like:
//
//	ALLOWED_IMPORT_PATHS = ["/tmp"]
//	TELEMETRY = "false"
//
// instead of (different order, which would fail the checkSecretModification)
//
//	TELEMETRY = "false"
//	ALLOWED_IMPORT_PATHS = ["/tmp"]
func sortKeys(a map[string]interface{}) []string {
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// addCustomPulpSettings appends custom settings defined in Pulp CR to settings.py
func addCustomPulpSettings(pulp *repomanagerpulpprojectorgv1beta2.Pulp, current_settings string) string {
	settings := pulp.Spec.PulpSettings.Raw
	var settingsJson map[string]interface{}
	json.Unmarshal(settings, &settingsJson)

	var convertedSettings string
	sortedKeys := sortKeys(settingsJson)
	for _, k := range sortedKeys {
		if strings.Contains(current_settings, strings.ToUpper(k)) {
			lines := strings.Split(current_settings, strings.ToUpper(k))
			current_settings = lines[0] + strings.Join(strings.Split(lines[1], "\n")[1:], "\n")
		}
		switch settingsJson[k].(type) {
		case map[string]interface{}:
			rawMapping, _ := json.Marshal(settingsJson[k])
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", strings.Replace(string(rawMapping), "\"", "'", -1))
		case []interface{}:
			rawMapping, _ := json.Marshal(settingsJson[k])
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", string(rawMapping))
		case bool:
			// Pulp expects True or False, but golang boolean values are true or false
			// so we are converting to string and changing to capital T or F
			convertToString := cases.Title(language.English, cases.Compact).String(strconv.FormatBool(settingsJson[k].(bool)))
			convertedSettings = convertedSettings + fmt.Sprintf("%v = %v\n", strings.ToUpper(k), convertToString)
		default:
			convertedSettings = convertedSettings + fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(k), settingsJson[k])
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

// checkSecretsAvailable verifies if the list of secrets that pulp-server secret can depend on
// are available.
func checkSecretsAvailable(funcResources controllers.FunctionResources) error {
	ctx := funcResources.Context
	pulp := funcResources.Pulp

	secrets := []string{"ObjectStorageAzureSecret", "ObjectStorageS3Secret", "SSOSecret"}
	for _, secretField := range secrets {
		structField := reflect.Indirect(reflect.ValueOf(pulp)).FieldByName("Spec").FieldByName(secretField)
		if structField.IsValid() && len(structField.Interface().(string)) != 0 {
			secret := &corev1.Secret{}
			if err := funcResources.Get(ctx, types.NamespacedName{Name: structField.Interface().(string), Namespace: pulp.Namespace}, secret); err != nil {
				return err
			}
		}
	}

	if len(pulp.Spec.Database.ExternalDBSecret) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.Database.ExternalDBSecret, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}

	if len(pulp.Spec.Cache.ExternalCacheSecret) != 0 {
		secret := &corev1.Secret{}
		if err := funcResources.Get(ctx, types.NamespacedName{Name: pulp.Spec.Cache.ExternalCacheSecret, Namespace: pulp.Namespace}, secret); err != nil {
			return err
		}
	}

	return nil
}

// checkImmutableFields verifies if a user tried to modify an immutable field and rollback
// the change if so
func (r *RepoManagerReconciler) checkImmutableFields(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, field immutableField, log logr.Logger) bool {

	fieldSpec := reflect.Value{}

	// access the field by its string name
	// for fieldSpec we need to pass it as a reference because we will need to change
	// its value back in case of immutable field
	switch field.FieldPath.(type) {
	case repomanagerpulpprojectorgv1beta2.PulpSpec:
		fieldSpec = reflect.Indirect(reflect.ValueOf(&pulp.Spec)).FieldByName(field.FieldName)
	case repomanagerpulpprojectorgv1beta2.Cache:
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
func (r *RepoManagerReconciler) updateIngressType(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {

	// if pulp CR was defined with route and user modified it to anything else
	// delete all routes with operator's labels
	// remove route .status.conditions
	// update .status.ingress_type = nodeport
	if strings.ToLower(pulp.Status.IngressType) == "route" && !isRoute(pulp) {

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

		// remove pulp-web components
		controllers.RemovePulpWebResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}

	// if pulp CR was defined with nodeport or loadbalancer and user modified it to anything else
	// delete all pulp-web resources
	// remove pulp-web .status.conditions
	// update .status.ingress_type
	// we will not remove configmap to avoid losing resources that are potentially unrecoverable
	if (strings.ToLower(pulp.Status.IngressType) == "nodeport" && strings.ToLower(pulp.Spec.IngressType) != "nodeport") ||
		(strings.ToLower(pulp.Status.IngressType) == "loadbalancer" && strings.ToLower(pulp.Spec.IngressType) != "loadbalancer") {

		// remove pulp-web components
		controllers.RemovePulpWebResources(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

		pulp.Status.IngressType = pulp.Spec.IngressType
		r.Status().Update(ctx, pulp)

		// nothing else to do (the controller will be responsible for setting up the other resources)
		return
	}
}

// updateIngressClass will check the current definition of ingress_class_name and will handle the different
// modification scenarios
func (r *RepoManagerReconciler) updateIngressClass(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {

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

	// handle OCP specific modifications on ingressclass change
	pulp_ocp.UpdateIngressClass(controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: nil, Logger: logr.Logger{}})

	pulp.Status.IngressClassName = pulp.Spec.IngressClassName
	r.Status().Update(ctx, pulp)
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
	*repomanagerpulpprojectorgv1beta2.Pulp
}

// createPulpResource executes a set of instructions to provision Pulp resources
func (r *RepoManagerReconciler) createPulpResource(resource ResourceDefinition, createFunc func(controllers.FunctionResources) client.Object) (bool, error) {
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
	case *corev1.ConfigMap:
		object = resourceType
		objKind = "ConfigMap"
	case *batchv1.CronJob:
		object = resourceType
		objKind = "CronJob"
	}

	// define the list of parameters to pass to "provisioner" function
	funcResources := controllers.FunctionResources{Context: resource.Context, Pulp: resource.Pulp, Logger: log, Scheme: r.Scheme, Client: r.Client}

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
		controllers.UpdateStatus(resource.Context, r.Client, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "Creating"+resource.Alias+objKind, "Creating "+resource.Name+" "+objKind)
		log.Info("Creating a new "+resource.Name+" "+objKind, "Namespace", resource.Pulp.Namespace, "Name", resource.Name)
		err = r.Create(resource.Context, expectedResource)

		// special condition for api deployments where we need to provide a warning message
		// in case no storage type is provided
		if resource.Name == resource.Pulp.Name+"-api" && objKind == "Deployment" {
			controllers.CheckEmptyDir(resource.Pulp, controllers.PulpResource)
		}

		if err != nil {
			log.Error(err, "Failed to create new "+resource.Name+" "+objKind)
			controllers.UpdateStatus(resource.Context, r.Client, resource.Pulp, metav1.ConditionFalse, resource.ConditionType, "ErrorCreating"+resource.Alias+objKind, "Failed to create "+resource.Name+" "+objKind+": "+err.Error())
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

// needsPulpWeb will return true if ingress_type is not route and the ingress_type provided does not
// support nginx controller, which is a scenario where pulp-web should be deployed
func (r *RepoManagerReconciler) needsPulpWeb(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return !isRoute(pulp) && !controllers.IsNginxIngressSupported(pulp)
}

// isNginxIngress will check if ingress_type is defined as "ingress"
func isIngress(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return strings.ToLower(pulp.Spec.IngressType) == "ingress"
}

// isRoute will check if ingress_type is defined as "route"
func isRoute(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return strings.ToLower(pulp.Spec.IngressType) == "route"
}

// isNginxIngress returns true if pulp is defined with ingress_type==ingress and the controller of the ingresclass provided is a nginx
func (r *RepoManagerReconciler) isNginxIngress(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return isIngress(pulp) && controllers.IsNginxIngressSupported(pulp)
}

// getRootURL handles user facing URLs
func getRootURL(resource controllers.FunctionResources) string {
	if isIngress(resource.Pulp) {
		hostname := resource.Pulp.Spec.IngressHost
		if len(resource.Pulp.Spec.Hostname) > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
			hostname = resource.Pulp.Spec.Hostname
		}
		return "https://" + hostname
	}
	if isRoute(resource.Pulp) {
		return "https://" + pulp_ocp.GetRouteHost(resource.Pulp)
	}

	return "http://" + resource.Pulp.Name + "-web-svc." + resource.Pulp.Namespace + ".svc.cluster.local:24880"
}

// ignoreUpdateCRStatusPredicate filters update events on pulpbackup CR status
func ignoreCronjobStatus() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*batchv1.CronJob)
			newObject := e.ObjectNew.(*batchv1.CronJob)

			// if old cronjob.status != new cronjob.status return false, which will instruct
			// the controller to do nothing on this update event
			return reflect.DeepEqual(oldObject.Status, newObject.Status)
		},
	}
}

// convertStringToMap is used to convert old ansible string fields (specifically annotations) into maps
// An example of usage is the service_annotation field, which is defined as string in ansible version,
// but the metadata.annotations is expecting map[string]string
func convertStringToMap(field string) map[string]string {
	convertedMap := map[string]string{}

	// using a bufio scanner to read the string line by line
	scanner := bufio.NewScanner(strings.NewReader(field))
	for scanner.Scan() {
		split := strings.Split(strings.TrimSpace(scanner.Text()), ":")
		// ignore empty fields
		if len(split) == 2 {
			convertedMap[split[0]] = split[1]
		}
	}

	return convertedMap
}

// ansibleMigrationTasks runs some instructions during upgrade from ansible to go version.
func ansibleMigrationTasks(resource controllers.FunctionResources) (ctrl.Result, error) {

	pulp := resource.Pulp
	log := resource.Logger
	ctx := resource.Context

	// We are using the DeployedImage field (available only in ansible) to verify if
	// the current CR comes from ansible.
	if len(pulp.Status.DeployedImage) == 0 || pulp.Status.MigrationDone {
		return ctrl.Result{}, nil
	}

	// Block with tasks specific to ansible -> go migration
	log.Info("Running migration tasks")

	// if .status.storagePersistentVolumeClaim is defined but .spec.pvc is not this is an upgrade from ansible version,
	// in this case, we need to update the PVC field to avoid the operator provisioning the resources with emptyDir.
	if len(pulp.Status.StoragePersistentVolumeClaim) > 0 && len(pulp.Spec.PVC) == 0 {
		log.Info("Updating spec.pvc field ...")
		if err := controllers.UpdateCRField(ctx, resource.Client, pulp, "PVC", pulp.Status.StoragePersistentVolumeClaim); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// in ansible version a PVC for the database is required (emptyDir is not an option)
	// since we are migrating, we need to deploy the new STS with the same PVC previously deployed.
	if len(pulp.Spec.Database.PVC) == 0 {
		log.Info("Updating spec.database.pvc field ...")
		oldPVC, err := getPostgresOldPVC(resource)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		patch := client.MergeFrom(pulp.DeepCopy())
		pulp.Spec.Database.PVC = oldPVC
		if err := resource.Client.Patch(ctx, pulp, patch); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// in ansible version Redis is always deployed
	if !pulp.Spec.Cache.Enabled {
		log.Info("Updating cache fields ...")
		patch := client.MergeFrom(pulp.DeepCopy())
		pulp.Spec.Cache.Enabled = true
		pulp.Spec.Cache.PVC = pulp.Name + "-redis-data"
		if err := resource.Client.Patch(ctx, pulp, patch); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// delete api/worker/content apis
	deleteOldAnsibleDeployments(resource)

	// scale old database sts to 0
	// we are not deleting it because golang version creates a new sts with different name and
	// users could manually delete it after reviewing the migration.
	scaleOldAnsibleSts(resource)

	// update old svc service to point to new postgres pods
	// (we will keep using it to avoid having to update the secrets with the new hostname)
	updateOldAnsibleSvc(resource)

	pulp.Status.MigrationDone = true
	resource.Client.Status().Update(ctx, pulp)
	log.Info("Migration tasks completed")
	return ctrl.Result{Requeue: true}, nil
}

// getOldPostgresLabels returns the list of labels used by ansible postgres sts, svc and pvc.
func getOldPostgresLabels(resource controllers.FunctionResources) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": "database",
		"app.kubernetes.io/instance":  "postgres-" + resource.Pulp.Name,
		"app.kubernetes.io/name":      "postgres",
	}
}

// getPostgresOldPVC returns the name of the Postgres PVC deployed by ansible operator
func getPostgresOldPVC(resource controllers.FunctionResources) (string, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	labels := getOldPostgresLabels(resource)
	listOpts := []client.ListOption{
		client.InNamespace(resource.Pulp.Namespace),
		client.MatchingLabels(labels),
	}
	if err := resource.Client.List(resource.Context, pvcList, listOpts...); err != nil {
		resource.Logger.Error(err, "Failed to retrieve the list of PostgreSQL volumes")
		return "", err
	}
	if len(pvcList.Items) == 0 {
		return "", fmt.Errorf("failed to find old ansible PostgreSQL volume")
	}
	return pvcList.Items[0].Name, nil
}

// deleteOldAnsibleDeployments removes old ansible deployments because they are not "compatible"
// with golang version, the label selectors (immutable fields) are different.
func deleteOldAnsibleDeployments(resource controllers.FunctionResources) {
	resource.Logger.Info("Removing old deployments ...")
	components := []string{"api", "content", "worker", "web", "redis"}
	for _, component := range components {
		deployment := &appsv1.Deployment{}
		deploymentName := resource.Pulp.Name + "-" + component
		if err := resource.Client.Get(resource.Context, types.NamespacedName{Namespace: resource.Pulp.Namespace, Name: deploymentName}, deployment); err != nil && !k8s_errors.IsNotFound(err) {
			resource.Logger.Error(err, "Failed to find old "+deploymentName+" deployment from ansible version.")
			continue
		}
		if err := resource.Client.Delete(resource.Context, deployment); err != nil {
			resource.Logger.Error(err, "Failed to remove old "+deploymentName+" deployment from ansible version.")
		}
	}
}

// scaleOldAnsibleSts scales old ansible postgres statefulset to 0 replicas
func scaleOldAnsibleSts(resource controllers.FunctionResources) {
	resource.Logger.Info("Scaling down old Postgres pods ...")
	replicas := int32(0)
	statefulSetList := &appsv1.StatefulSetList{}
	labels := getOldPostgresLabels(resource)
	listOpts := []client.ListOption{
		client.InNamespace(resource.Pulp.Namespace),
		client.MatchingLabels(labels),
	}
	if err := resource.Client.List(resource.Context, statefulSetList, listOpts...); err != nil {
		resource.Logger.Error(err, "Failed to retrieve the list of old ansible StatefulSets.")
		return
	}

	statefulSet := &appsv1.StatefulSet{}
	statefulSetName := statefulSetList.Items[0].Name
	if err := resource.Client.Get(resource.Context, types.NamespacedName{Namespace: resource.Pulp.Namespace, Name: statefulSetName}, statefulSet); err != nil {
		resource.Logger.Error(err, "Failed to find old "+statefulSetName+" StatefulSet from ansible version.")
		return
	}

	patch := client.MergeFrom(statefulSet.DeepCopy())
	statefulSet.Spec.Replicas = &replicas
	if err := resource.Client.Patch(resource.Context, statefulSet, patch); err != nil {
		resource.Logger.Error(err, "Failed to scale down old "+statefulSetName+" StatefulSet from ansible version.")
	}
}

// updateOldAnsibleSvc changes the label selector from ansible postgres service.
func updateOldAnsibleSvc(resource controllers.FunctionResources) {
	resource.Logger.Info("Updating Postgres service ...")
	serviceList := &corev1.ServiceList{}
	labels := getOldPostgresLabels(resource)
	listOpts := []client.ListOption{
		client.InNamespace(resource.Pulp.Namespace),
		client.MatchingLabels(labels),
	}
	if err := resource.Client.List(resource.Context, serviceList, listOpts...); err != nil {
		resource.Logger.Error(err, "Failed to retrieve the list of old ansible Services.")
		return
	}

	service := &corev1.Service{}
	serviceName := serviceList.Items[0].Name
	if err := resource.Client.Get(resource.Context, types.NamespacedName{Namespace: resource.Pulp.Namespace, Name: serviceName}, service); err != nil {
		resource.Logger.Error(err, "Failed to find old "+serviceName+" Service from ansible version.")
		return
	}

	newLabels := map[string]string{
		"app":     "postgresql",
		"pulp_cr": resource.Pulp.Name,
	}
	maps.Copy(newLabels, labels)
	patch := client.MergeFrom(service.DeepCopy())
	service.Spec.Selector = newLabels
	if err := resource.Client.Patch(resource.Context, service, patch); err != nil {
		resource.Logger.Error(err, "Failed to change selectors from old "+serviceName+" Service ansible version.")
	}
}

// findPulpDependentSecrets will search for Pulp objects based on Secret names defined in Pulp CR.
// It is used to "link" these Secrets (not "owned" by Pulp operator) with Pulp object
func (r *RepoManagerReconciler) findPulpDependentSecrets(secret client.Object) []reconcile.Request {

	associatedPulp := repomanagerpulpprojectorgv1beta2.PulpList{}
	opts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("secrets", secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	if err := r.List(context.TODO(), &associatedPulp, opts); err != nil {
		return []reconcile.Request{}
	}
	if len(associatedPulp.Items) > 0 {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      associatedPulp.Items[0].GetName(),
				Namespace: associatedPulp.Items[0].GetNamespace(),
			},
		}}
	}

	return []reconcile.Request{}
}

// restartPods modifies a deployment template field (`.annotations`) which will
// start a new rollout of pods
func (r *RepoManagerReconciler) restartPods(pulp *repomanagerpulpprojectorgv1beta2.Pulp, obj client.Object) {
	switch obj := obj.(type) {
	case *appsv1.Deployment:

		patch := client.MergeFrom(obj.DeepCopy())
		if obj.Spec.Template.ObjectMeta.Annotations == nil {
			obj.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		obj.Spec.Template.ObjectMeta.Annotations["repo-manager.pulpproject.org/restartedAt"] = time.Now().Format(time.RFC3339)
		r.Patch(context.TODO(), obj, patch)
	}
}

// runMigration deploys a k8s Job to run django migrations in case of pulpcore image change
func (r *RepoManagerReconciler) runMigration(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	if !r.needsMigration(ctx, pulp) {
		return
	}
	r.migrationJob(ctx, pulp)
}

// needsMigration verifies if the pulpcore image has changed and no migration
// has been done yet.
func (r *RepoManagerReconciler) needsMigration(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return controllers.ImageChanged(pulp) && !r.migrationDone(ctx, pulp)
}

// migrationDone checks if there is a migration Job with the expected image
func (r *RepoManagerReconciler) migrationDone(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	jobList := &batchv1.JobList{}
	labels := jobLabels(*pulp)
	listOpts := []client.ListOption{
		client.InNamespace(pulp.Namespace),
		client.MatchingLabels(labels),
	}

	r.List(ctx, jobList, listOpts...)
	return hasActiveJob(*jobList, pulp)
}

// jobActive returns true if there is at least one running pod for Job.
// This is a workaround to identify if the Job is still running because, as of now,
// it is not possible to filter - using client.ListOption - the list of Jobs using
// something like client.MatchingFieldsSelector{ "state.running=true" }
func jobActive(job batchv1.Job) bool {
	return job.Status.Active >= 1
}

// jobImageEqualsCurrent verifies if the image used in migration job is the same
// as the one used in pulpcore-{api,content,worker} pods
func jobImageEqualsCurrent(job batchv1.Job, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return job.Spec.Template.Spec.Containers[0].Image == pulp.Spec.Image+":"+pulp.Spec.ImageVersion
}

// hasActiveJob will iterate over the JobList looking for any Job with the current
// pulpcore image (meaning that a migration for the current version has already
// been triggered and there is no need to create a new job).
func hasActiveJob(jobList batchv1.JobList, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	for _, job := range jobList.Items {
		if jobImageEqualsCurrent(job, pulp) {
			return true
		}
	}
	return false
}
