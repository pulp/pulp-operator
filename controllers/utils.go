/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	AzureObjType = "azure blob"
	S3ObjType    = "s3"
	SCNameType   = "StorageClassName"
	PVCType      = "PVC"
	EmptyDirType = "emptyDir"

	PulpResource     = "Pulp"
	CacheResource    = "Cache"
	DatabaseResource = "Database"

	DotNotEditMessage = `
# This file is managed by Pulp operator.
# DO NOT EDIT IT.
#
# To modify custom fields, use the pulp_settings from Pulp CR, for example:
# spec:
#   pulp_settings:
#     allowed_export_paths:
#     - /tmp

`
	DefaultOCPIngressClass = "openshift-default"
)

// FunctionResources contains the list of arguments passed to create new Pulp resources
type FunctionResources struct {
	context.Context
	client.Client
	*repomanagerpulpprojectorgv1beta2.Pulp
	Scheme *runtime.Scheme
	logr.Logger
}

// Deployer is an interface for the several deployment types:
// - api deployment in vanilla k8s or ocp
// - content deployment in vanilla k8s or ocp
// - worker deployment in vanilla k8s or ocp
type Deployer interface {
	deploy() client.Object
}

// IgnoreUpdateCRStatusPredicate filters update events on pulpbackup CR status
func IgnoreUpdateCRStatusPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}

// IsOpenShift returns true if the platform cluster is OpenShift
func IsOpenShift() (bool, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return false, err
	}
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}

	_, err = client.ServerResourcesForGroupVersion("config.openshift.io/v1")

	if err != nil && k8s_errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// MultiStorageConfigured returns true if Pulp CR is configured with more than one "storage type"
// for example, if ObjectStorageAzureSecret and FileStorageClass are defined we can't determine
// which one the operator should use.
func MultiStorageConfigured(pulp *repomanagerpulpprojectorgv1beta2.Pulp, resource string) (bool, []string) {
	var names []string

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// for ansible migration we are ignoring the multistorage check
	if pulp.Status.MigrationDone {
		if resource == DatabaseResource {
			return false, []string{PVCType}
		}
		if len(pulp.Spec.ObjectStorageAzureSecret) > 0 {
			return false, []string{AzureObjType}
		}
		if len(pulp.Spec.ObjectStorageS3Secret) > 0 {
			return false, []string{S3ObjType}
		}
		return false, []string{PVCType}
	}

	switch resource {
	case PulpResource:
		if len(pulp.Spec.ObjectStorageAzureSecret) > 0 {
			names = append(names, AzureObjType)
		}

		if len(pulp.Spec.ObjectStorageS3Secret) > 0 {
			names = append(names, S3ObjType)
		}

		if len(pulp.Spec.FileStorageClass) > 0 {
			names = append(names, SCNameType)
		}

		if len(pulp.Spec.PVC) > 0 {
			names = append(names, PVCType)
		}

		if len(names) > 1 {
			return true, names
		} else if len(names) == 0 {
			return false, []string{EmptyDirType}
		}

	case CacheResource:
		if len(pulp.Spec.Cache.RedisStorageClass) > 0 {
			names = append(names, SCNameType)
		}

		if len(pulp.Spec.Cache.PVC) > 0 {
			names = append(names, PVCType)
		}

		if len(names) > 1 {
			return true, names
		} else if len(names) == 0 {
			return false, []string{EmptyDirType}
		}

	case DatabaseResource:
		if len(pulp.Spec.Database.PVC) > 0 {
			names = append(names, PVCType)
		}
		if pulp.Spec.Database.PostgresStorageClass != nil {
			names = append(names, SCNameType)
		}

		if len(names) > 1 {
			return true, names
		} else if len(names) == 0 {
			return false, []string{EmptyDirType}
		}
	}

	return false, names
}

// ContainerExec runs a command in the container
func ContainerExec[T any](client T, pod *corev1.Pod, command []string, container, namespace string) (string, error) {

	// get the concrete value of client ({PulpBackup,RepoManagerBackupReconciler,RepoManagerRestoreReconciler})
	clientConcrete := reflect.ValueOf(client)

	// here we are using the Indirect method to get the value where client is pointing to
	// after that we are taking the RESTClient field from PulpBackup|RepoManagerBackupReconciler|RepoManagerRestoreReconciler and
	// "transforming" it into an interface{} (through the Interface() method)
	// and finally we are asserting that it is a *rest.RESTClient so that we can run the Post() method later
	restClient := reflect.Indirect(clientConcrete).FieldByName("RESTClient").Elem().Interface().(*rest.RESTClient)

	// we are basically doing the same as before, but this time asserting as runtime.Scheme and rest.Config
	runtimeScheme := reflect.Indirect(clientConcrete).FieldByName("Scheme").Elem().Interface().(runtime.Scheme)
	restConfig := reflect.Indirect(clientConcrete).FieldByName("RESTConfig").Elem().Interface().(rest.Config)

	execReq := restClient.
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, runtime.NewParameterCodec(&runtimeScheme))

	exec, err := remotecommand.NewSPDYExecutor(&restConfig, "POST", execReq.URL())
	if err != nil {
		return "", err
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String())
	result = strings.TrimSpace(result)

	// [TODO] remove this sleep and find a better way to make sure that it finished execution
	// I think the exec.Stream command is not synchronous and sometimes when a task depends
	// on the results of the previous one it is failing.
	// But this is just a guess!!! We need to investigate it further.
	time.Sleep(time.Second)
	return result, nil
}

// IsNginxIngressSupported returns true if the operator was instructed that this is a nginx ingress controller
func IsNginxIngressSupported(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	return pulp.Spec.IsNginxIngress
}

// CustomZapLogger should be used only for warn messages
// it is a kludge to bypass the "limitation" of logr not having warn level
func CustomZapLogger() *zap.Logger {
	econdeTime := func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339))
	}
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.WarnLevel),
		Development:      false,
		OutputPaths:      []string{"stdout"},
		Encoding:         "console",
		ErrorOutputPaths: []string{"stderr"},
		DisableCaller:    false,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "logger",
			CallerKey:      "file",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     econdeTime,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
	}

	logger, _ := cfg.Build()
	defer logger.Sync()

	return logger
}

// CheckEmptyDir creates a log warn message in case no persistent storage is provided
// for the given resource
func CheckEmptyDir(pulp *repomanagerpulpprojectorgv1beta2.Pulp, resource string) {
	_, storageType := MultiStorageConfigured(pulp, resource)
	if storageType[0] == EmptyDirType {
		logger := CustomZapLogger()
		logger.Warn("No StorageClass or PVC defined for " + strings.ToUpper(resource) + " pods!")
		logger.Warn("CONFIGURING " + strings.ToUpper(resource) + " POD VOLUME AS EMPTYDIR. THIS SHOULD NOT BE USED IN PRODUCTION CLUSTERS.")
	}
}

// CheckImageVersionModified verifies if the container image tag defined in
// Pulp CR matches the one in the Deployment
func CheckImageVersionModified(pulp *repomanagerpulpprojectorgv1beta2.Pulp, deployment *appsv1.Deployment) bool {
	r := regexp.MustCompile(`(?P<ImageName>.*?):(?P<Tag>.*)`)
	currentImageVersion := r.FindStringSubmatch(deployment.Spec.Template.Spec.Containers[0].Image)
	return pulp.Spec.ImageVersion != currentImageVersion[2]
}

// WaitAPIPods waits until all API pods are in a READY state
func WaitAPIPods[T any](resource T, pulp *repomanagerpulpprojectorgv1beta2.Pulp, deployment *appsv1.Deployment, timeout time.Duration) {

	// we need to add a litte "stand by" to give time for the operator get the updated status from database/cluster
	time.Sleep(time.Millisecond * 500)
	clientConcrete := reflect.ValueOf(resource)
	restClient := reflect.Indirect(clientConcrete).FieldByName("Client").Elem().Interface().(client.Client)
	for i := 0; i < int(timeout.Seconds()); i++ {
		apiDeployment := &appsv1.Deployment{}
		restClient.Get(context.TODO(), types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, apiDeployment)
		if apiDeployment.Status.ReadyReplicas == apiDeployment.Status.Replicas {
			return
		}
		time.Sleep(time.Second)
	}
}
func GetPulpSetting(pulp *repomanagerpulpprojectorgv1beta2.Pulp, key string) string {
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
			domainEnabled := settingsJson[strings.ToLower("domain_enabled")]
			if domainEnabled != nil && domainEnabled.(bool) {
				return "/pulp/content/default/"
			}
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

// getSigningKeyFingerprint returns the signing key fingerprint from secret object
func getSigningKeyFingerprint(r client.Client, secretName, secretNamespace string) (string, error) {

	ctx := context.TODO()
	secretData, err := RetrieveSecretData(ctx, secretName, secretNamespace, true, r, "signing_service.gpg")
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

// Retrieve specific keys from secret object
func RetrieveSecretData(ctx context.Context, secretName, secretNamespace string, required bool, r client.Client, keys ...string) (map[string]string, error) {
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

// UpdateStatus will set the new condition value for a .status.conditions[]
// it will also set Pulp-Operator-Finished-Execution to false
func UpdateStatus(ctx context.Context, r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp, conditionStatus metav1.ConditionStatus, conditionType, conditionReason, conditionMessage string) {

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

// checkSpecModification returns true if .spec fields present on A are equal to
// what is in B
func checkSpecModification(currentField, expectedField interface{}) bool {
	return !equality.Semantic.DeepDerivative(currentField, expectedField)
}

// CheckDeployment returns true if a spec from deployment is not
// with the expected contents defined in Pulp CR
func CheckDeploymentSpec(expectedState, currentState interface{}) bool {

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

// checkPulpServerSecretModification returns true if the settings.py from pulp-server secret
// does not have the expected contents
func checkPulpServerSecretModification(expectedState, currentState interface{}) bool {
	expectedData := expectedState.(map[string]string)["settings.py"]
	currentData := string(currentState.(map[string][]byte)["settings.py"])
	return expectedData != currentData
}

// [TODO] Pending implementation. Not sure if we should focus on this now considering
// that for production clusters an external postgres should be used.
func checkPostgresSecretModification(expectedState, currentState interface{}) bool {
	return false
}

// isRouteOrIngress is used to assert if the provided resource is an ingress or a route
func isRouteOrIngress(resource interface{}) bool {
	_, route := resource.(*routev1.Route)
	_, ingress := resource.(*netv1.Ingress)
	return route || ingress
}

// updateObject is a function that verifies if an object has been modified
// and update, if necessary, with the expected config
func updateObject(resources FunctionResources, modified func(interface{}, interface{}) bool, objKind, conditionType, field string, expectedState, currentState client.Object) (bool, error) {

	log := resources.Logger
	client := resources.Client
	pulp := resources.Pulp

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
	} else if field == "Data" {
		currentField = reflect.Indirect(reflect.ValueOf(currentState)).FieldByName(field).Interface()
		if objKind == "Secret" {
			// Retrieving a Secret using k8s-client will return the Data field (map[string][uint8])
			// When we are creating the secrets (for example, through the pulpServerSecret function) we are
			// using the StringData field
			expectedField = reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName("StringData").Interface()
		} else {
			expectedField = reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName("Data").Interface()
		}
	}

	if modified(expectedField, currentField) {
		log.Info("The " + field + " from " + objKind + " " + objName + " has been modified! Reconciling ...")
		UpdateStatus(resources.Context, client, pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)

		// for ingress and/or routes we need to "manually" update ResourceVersion fields
		// to avoid the issue
		//    "the object has been modified; please apply your changes to the latest version and try again"
		if isRouteOrIngress(expectedState) {
			reflect.ValueOf(expectedState).MethodByName("SetResourceVersion").Call(reflect.ValueOf(currentState).MethodByName("GetResourceVersion").Call([]reflect.Value{}))
		}
		if err := client.Update(resources.Context, expectedState); err != nil {
			log.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
			UpdateStatus(resources.Context, client, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// ReconcileObject will check if the definition from Pulp CR is reflecting the current
// object state and if not will synchronize the configuration
func ReconcileObject(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string) (bool, error) {

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
		// we need to do this check only for pulp-web-svc service because it is
		// the only nodePort svc
		if expectedState.GetName() == funcResources.Pulp.Name+"-web-svc" && funcResources.Pulp.Spec.NodePort == 0 {
			return false, nil
		}
	case *appsv1.Deployment:
		objKind = "Deployment"
		checkFunction = CheckDeploymentSpec
	case *corev1.ConfigMap:
		objKind = "ConfigMap"
		return updateObject(funcResources, checkFunction, objKind, conditionType, "Data", expectedState, currentState)
	case *corev1.Secret:
		// by default, defines secretModFunc with the function to verify if pulp-server secret has been modified
		secretModFunc := checkPulpServerSecretModification

		// if the secret to be verified is the postgres-configuration we need to use another function
		if expectedState.GetName() == funcResources.Pulp.Name+"-postgres-configuration" {
			secretModFunc = checkPostgresSecretModification
		}
		return updateObject(funcResources, secretModFunc, "Secret", conditionType, "Data", expectedState, currentState)
	default:
		return false, nil
	}

	return updateObject(funcResources, checkFunction, objKind, conditionType, "Spec", expectedState, currentState)
}

// checkMetadataModification returns true if annotations or labels fields are not equal
// it is used to compare .metadata.annotations and .metadata.labels fields
// since these are map[string]string types
// we are using equality.Semantic.DeepEqual for both cases
//
//	(currentField, expectedField) and (expectedField, currentField)
//
// to evaluate map[string]nil == map[string]""(empty string) and
// map[string]"" == map[string]nil
func checkMetadataModification(currentField, expectedField interface{}) bool {
	return !equality.Semantic.DeepEqual(currentField, expectedField) || !equality.Semantic.DeepEqual(expectedField, currentField)
}

// ReconcileMetadata is a method to handle only .metadata.{labels,annotations} reconciliation
// for some reason, if we try to use DeepDerivative like
//
//	if !equality.Semantic.DeepDerivative(expectedState.(*routev1.Route), currentState.(*routev1.Route)) ...
//
// it will get into an infinite loop
func ReconcileMetadata(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string) (bool, error) {

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

// UpdatCRField patches fieldName in Pulp CR with fieldValue
func UpdateCRField(ctx context.Context, r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp, fieldName, fieldValue string) error {
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

// RemovePulpWebResources deletes pulp-web Deployment and Service
func RemovePulpWebResources(resources FunctionResources) error {
	ctx := resources.Context
	pulp := resources.Pulp

	// remove pulp-web components
	webDeployment := &appsv1.Deployment{}
	if err := resources.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web", Namespace: pulp.Namespace}, webDeployment); err == nil {
		resources.Delete(ctx, webDeployment)
	} else {
		return err
	}

	webSvc := &corev1.Service{}
	if err := resources.Get(ctx, types.NamespacedName{Name: pulp.Name + "-web-svc", Namespace: pulp.Namespace}, webSvc); err == nil {
		resources.Delete(ctx, webSvc)
	} else {
		return err
	}

	webConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"
	v1.RemoveStatusCondition(&pulp.Status.Conditions, webConditionType)

	return nil
}
