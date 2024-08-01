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
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers/settings"
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
	"k8s.io/apimachinery/pkg/util/dump"
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
	SCNameType   = "StorageClass"
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
	OperatorHashLabel      = "pulp-operator-hash"
)

// FunctionResources contains the list of arguments passed to create new Pulp resources
type FunctionResources struct {
	context.Context
	client.Client
	*repomanagerpulpprojectorgv1beta2.Pulp
	Scheme *runtime.Scheme
	logr.Logger
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
func ImageChanged(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	definedImage := pulp.Spec.Image + ":" + pulp.Spec.ImageVersion
	currentImage := pulp.Status.Image
	return currentImage != definedImage
}

// StorageTypeChanged verifies if the storage type has been modified
func StorageTypeChanged(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	currentStorageType := pulp.Status.StorageType
	definedStorageType := GetStorageType(*pulp)
	return currentStorageType != definedStorageType[0]
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

// getPulpSetting returns the value of a Pulp setting field
func getPulpSetting(r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp, key string) string {
	// [DEPRECATED] PulppSettings should not be used anymore. Keeping it to avoid compatibility issues
	if settings := pulp.Spec.PulpSettings.Raw; settings != nil {
		var settingsJson map[string]interface{}
		json.Unmarshal(settings, &settingsJson)
		if setting := settingsJson[key]; setting != nil && setting.(string) != "" {
			return setting.(string)
		}
	}

	if pulp.Spec.CustomPulpSettings != "" {
		settingsCM := &corev1.ConfigMap{}
		r.Get(context.TODO(), types.NamespacedName{Name: pulp.Spec.CustomPulpSettings, Namespace: pulp.Namespace}, settingsCM)
		if setting, found := settingsCM.Data[key]; found {
			return setting
		}
	}

	return ""
}

// domainEnabled returns the definition of DOMAIN_ENABLED in settings.py
func domainEnabled(r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	if domainEnabled := getPulpSetting(r, pulp, "domain_enabled"); domainEnabled != "" {
		enabled, _ := strconv.ParseBool(domainEnabled)
		return enabled
	}
	return false
}

// GetAPIRoot returns the definition of API_ROOT in settings.py or /pulp/
func GetAPIRoot(r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	if apiRoot := getPulpSetting(r, pulp, "api_root"); apiRoot != "" {
		return apiRoot
	}
	return "/pulp/"
}

// GetContentPathPrefix returns the definition of CONTENT_PATH_PREFIX in settings.py or
// * /pulp/content/default/ if domain is enabled
// * /pulp/content/ otherwise
func GetContentPathPrefix(r client.Client, pulp *repomanagerpulpprojectorgv1beta2.Pulp) string {
	if contentPath := getPulpSetting(r, pulp, "content_path_prefix"); contentPath != "" {
		return contentPath
	}

	if domainEnabled(r, pulp) {
		return "/pulp/content/default/"
	}
	return "/pulp/content/"
}

// getSigningKeyFingerprint returns the signing key fingerprint from secret object
func GetSigningKeyFingerprint(r client.Client, secretName, secretNamespace string) (string, error) {

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
func checkSpecModification(fields ...interface{}) bool {
	expectedField := fields[0]
	currentField := fields[1]
	return !equality.Semantic.DeepDerivative(expectedField, currentField)
}

// CheckDeployment returns true if a spec from deployment is not
// with the expected contents defined in Pulp CR
func CheckDeploymentSpec(fields ...interface{}) bool {
	expected := fields[0].(appsv1.Deployment)
	current := fields[1].(appsv1.Deployment)
	hashFromLabel := GetCurrentHash(&current)
	hashFromExpected := HashFromMutated(&expected, fields[2].(FunctionResources))
	hashFromCurrent := CalculateHash(current.Spec)
	return deploymentChanged(hashFromLabel, hashFromExpected, hashFromCurrent)
}

// deploymentChanged returns true if
// * the hash stored as deployment label is equal to the hash calculated from the expected spec AND
// * the hash calculated from the current deployment spec is equal to the hash calculated from the expected spec
func deploymentChanged(hashFromLabel, hashFromExpected, hashFromCurrent string) bool {
	return hashFromLabel != hashFromExpected || hashFromCurrent != hashFromExpected
}

// checkPulpServerSecretModification returns true if the settings.py from pulp-server secret
// does not have the expected contents
func checkPulpServerSecretModification(fields ...interface{}) bool {
	expectedData := CalculateHash(fields[0])
	convertCurrent := convertSecretDataFormat(fields[1].(map[string][]byte))
	currentData := CalculateHash(convertCurrent)
	return expectedData != currentData
}

// convertSecretDataFormat converts the Secret's data field into a map[string]string
func convertSecretDataFormat(secret map[string][]byte) map[string]string {
	converted := map[string]string{}
	for k := range secret {
		converted[k] = string(secret[k])
	}
	return converted
}

// isRouteOrIngress is used to assert if the provided resource is an ingress or a route
func isRouteOrIngress(resource interface{}) bool {
	_, route := resource.(*routev1.Route)
	_, ingress := resource.(*netv1.Ingress)
	return route || ingress
}

// PulpObject represents Pulp resources managed by pulp-operator
type PulpObject interface {
	GetFields(...interface{}) []interface{}
	GetModifiedFunc() func(...interface{}) bool
	GetFieldAndKind() (string, string)
}
type PulpDeployment struct{}
type PulpSecret struct{}
type PulpService struct{}
type PulpConfigMap struct{}
type PulpIngress struct{}
type PulpRoute struct{}
type PulpObjectMetadata struct{}

// GetFields expects 3 arguments:
// * the current Deployment spec
// * the expected Deployment spec
// * a FunctionResources so that CheckDeploymentSpec can use it to get the labels from current Deployment
func (PulpDeployment) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedSpec := reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).Interface()
	currentSpec := reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).Interface()
	return append(fieldsState, expectedSpec, currentSpec, obj[2].(FunctionResources))
}

// GetModifiedFunc returns the function used to check the Deployment modification
func (PulpDeployment) GetModifiedFunc() func(...interface{}) bool {
	return CheckDeploymentSpec
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpDeployment) GetFieldAndKind() (string, string) {
	return "Spec", "Deployment"
}

// GetFields expects 2 arguments:
// * the current encoded Secret data
// * the expected "raw" Secret string data
func (PulpSecret) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedData := reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("StringData").Interface()
	currentData := reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("Data").Interface()
	return append(fieldsState, expectedData, currentData)
}

// GetModifiedFunc returns the function used to check the Secret modification
func (PulpSecret) GetModifiedFunc() func(...interface{}) bool {
	return checkPulpServerSecretModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpSecret) GetFieldAndKind() (string, string) {
	return "Data", "Secret"
}

// GetFields expects 2 arguments:
// * the current ConfigMap data
// * the expected ConfigMap data
func (PulpConfigMap) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedData := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("Data").Interface())
	currentData := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("Data").Interface())
	return append(fieldsState, expectedData, currentData)
}

// GetModifiedFunc returns the function used to check the COnfigMap modification
func (PulpConfigMap) GetModifiedFunc() func(...interface{}) bool {
	return checkSpecModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpConfigMap) GetFieldAndKind() (string, string) {
	return "Data", "ConfigMap"
}

// GetFields expects 2 arguments:
// * the current Service spec field
// * the expected Service spec field
func (PulpService) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("Spec").Interface())
	currentSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("Spec").Interface())
	return append(fieldsState, expectedSpec, currentSpec)
}

// GetModifiedFunc returns the function used to check the Service modification
func (PulpService) GetModifiedFunc() func(...interface{}) bool {
	return checkSpecModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpService) GetFieldAndKind() (string, string) {
	return "Spec", "Service"
}

// GetFields expects 2 arguments:
// * the current Service spec field
// * the expected Service spec field
func (PulpIngress) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("Spec").Interface())
	currentSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("Spec").Interface())
	return append(fieldsState, expectedSpec, currentSpec)
}

// GetModifiedFunc returns the function used to check the Ingress modification
func (PulpIngress) GetModifiedFunc() func(...interface{}) bool {
	return checkSpecModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpIngress) GetFieldAndKind() (string, string) {
	return "Spec", "Ingress"
}

// GetFields expects 3 arguments:
// * the current Deployment spec
// * the expected Deployment spec
// * a FunctionResources so that CheckDeploymentSpec can use it to get the labels from current Deployment
func (PulpRoute) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("Spec").Interface())
	currentSpec := append(fieldsState, reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("Spec").Interface())
	return append(fieldsState, expectedSpec, currentSpec)
}

// GetModifiedFunc returns the function used to check the Route modification
func (PulpRoute) GetModifiedFunc() func(...interface{}) bool {
	return checkSpecModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpRoute) GetFieldAndKind() (string, string) {
	return "Spec", "Route"
}

// GetFields expects 2 arguments:
// * the current Object definition
// * the expected Object definition
func (PulpObjectMetadata) GetFields(obj ...interface{}) []interface{} {
	var fieldsState []interface{}
	expectedState := reflect.Indirect(reflect.ValueOf(obj[0].(client.Object))).FieldByName("ObjectMeta").Interface()
	currentState := reflect.Indirect(reflect.ValueOf(obj[1].(client.Object))).FieldByName("ObjectMeta").Interface()
	return append(fieldsState, expectedState, currentState)
}

// GetModifiedFunc returns the function used to check the Metadata modification
func (PulpObjectMetadata) GetModifiedFunc() func(...interface{}) bool {
	return checkMetadataModification
}

// GetFieldAndKind returns the field being checked and the object kind
func (PulpObjectMetadata) GetFieldAndKind() (string, string) {
	return "Metadata", "pulp resource"
}

// updateObject is a function that verifies if an object has been modified
// and update, if necessary, with the expected config
func updateObject(resources FunctionResources, modified func(...interface{}) bool, objKind, conditionType, field string, expectedState, currentState client.Object, pulpObject PulpObject) (bool, error) {

	log := resources.Logger
	client := resources.Client
	pulp := resources.Pulp

	// get object name
	objName := reflect.Indirect(reflect.ValueOf(expectedState)).FieldByName("Name").Interface().(string)

	// consolidate the fields to be verified/compared into a slice
	fieldsState := pulpObject.GetFields(expectedState, currentState, resources)

	if modified(fieldsState...) {
		log.Info("The " + field + " from " + objKind + " " + objName + " has been modified! Reconciling ...")
		UpdateStatus(resources.Context, client, pulp, metav1.ConditionFalse, conditionType, "Updating"+objKind, "Reconciling "+objName+" "+objKind)

		// for ingress and/or routes we need to "manually" update ResourceVersion fields
		// to avoid the issue
		//    "the object has been modified; please apply your changes to the latest version and try again"
		if isRouteOrIngress(expectedState) {
			reflect.ValueOf(expectedState).MethodByName("SetResourceVersion").Call(reflect.ValueOf(currentState).MethodByName("GetResourceVersion").Call([]reflect.Value{}))
		}
		if err := client.Update(resources.Context, expectedState); err != nil && !k8s_errors.IsConflict(err) {
			log.Error(err, "Error trying to update "+objName+" "+objKind+" ...")
			UpdateStatus(resources.Context, client, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdating"+objKind, "Failed to reconcile "+objName+" "+objKind+": "+err.Error())
			return false, err
		} else if err != nil && k8s_errors.IsConflict(err) {
			// whenever we get the "**object has been modified**" error we can just
			// trigger a new reconciliation to get the updated object and try again
			return true, nil
		}
		return true, nil
	}
	return false, nil
}

// ReconcileObject will check if the definition from Pulp CR is reflecting the current
// object state and if not will synchronize the configuration
// func ReconcileObject(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string, pulpObject PulpObject) (bool, error) {
func ReconcileObject(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string, pulpObject PulpObject) (bool, error) {

	// if NodePort field is REMOVED we dont need to do anything
	// kubernetes will define a new nodeport automatically
	// we need to do this check only for pulp-web-svc service because it is
	// the only nodePort svc (this is an edge case)
	if expectedState.GetName() == settings.PulpWebService(funcResources.Pulp.Name) && funcResources.Pulp.Spec.NodePort == 0 {
		return false, nil
	}

	field, objKind := pulpObject.GetFieldAndKind()
	return updateObject(funcResources, pulpObject.GetModifiedFunc(), objKind, conditionType, field, expectedState, currentState, pulpObject)
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
func checkMetadataModification(fields ...interface{}) bool {
	return metadataAnnotationsDiff(fields...) || metadataLabelsDiff(fields...)
}

// metadataAnnotationsDiff returns true if the expected annotations are diff from the current
func metadataAnnotationsDiff(fields ...interface{}) bool {
	currentAnnotations := fields[0].(metav1.ObjectMeta).Annotations
	expectedAnnotations := fields[1].(metav1.ObjectMeta).Annotations
	return !equality.Semantic.DeepEqual(currentAnnotations, expectedAnnotations) || !equality.Semantic.DeepEqual(expectedAnnotations, currentAnnotations)
}

// metadataLabelsDiff returns true if the expected labels are diff from the current
func metadataLabelsDiff(fields ...interface{}) bool {
	currentLabels := fields[0].(metav1.ObjectMeta).Labels
	expectedLabels := fields[1].(metav1.ObjectMeta).Labels
	return !equality.Semantic.DeepEqual(currentLabels, expectedLabels) || !equality.Semantic.DeepEqual(expectedLabels, currentLabels)
}

// ReconcileMetadata is a method to handle only .metadata.{labels,annotations} reconciliation
// for some reason, if we try to use DeepDerivative like
//
//	if !equality.Semantic.DeepDerivative(expectedState.(*routev1.Route), currentState.(*routev1.Route)) ...
//
// it will get into an infinite loop
func ReconcileMetadata(funcResources FunctionResources, expectedState, currentState client.Object, conditionType string) (bool, error) {
	pulpObject := PulpObjectMetadata{}
	field, objKind := pulpObject.GetFieldAndKind()
	return updateObject(funcResources, checkMetadataModification, objKind, conditionType, field, expectedState, currentState, pulpObject)
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
	if err := resources.Get(ctx, types.NamespacedName{Name: settings.WEB.DeploymentName(pulp.Name), Namespace: pulp.Namespace}, webDeployment); err == nil {
		resources.Delete(ctx, webDeployment)
	} else {
		return err
	}

	webSvc := &corev1.Service{}
	if err := resources.Get(ctx, types.NamespacedName{Name: settings.PulpWebService(pulp.Name), Namespace: pulp.Namespace}, webSvc); err == nil {
		resources.Delete(ctx, webSvc)
	} else {
		return err
	}

	webConditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Web-Ready"
	v1.RemoveStatusCondition(&pulp.Status.Conditions, webConditionType)

	return nil
}

// CalculateHash returns a string of the hashed value from obj
func CalculateHash(obj any) string {
	calculatedHash := fnv.New32a()
	// This is equivalent to: `k8s.io/kubernetes/pkg/util/hash.DeepHashObject(calculatedHash, obj)`
	// but avoids depending on k8s.io/kubernetes: https://github.com/kubernetes/kubernetes/issues/79384
	_, _ = fmt.Fprintf(calculatedHash, "%v", dump.ForHash(obj))
	return hex.EncodeToString(calculatedHash.Sum(nil))
}

// SetHashLabel appends the operator's hash label into object
func SetHashLabel(label string, obj client.Object) {
	currentLabels := obj.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}
	currentLabels[OperatorHashLabel] = label
	obj.SetLabels(currentLabels)
}

// getCurrentHash retrieves the hash defined in obj label
func GetCurrentHash(obj client.Object) string {
	return obj.GetLabels()[OperatorHashLabel]
}

func HashFromMutated(dep *appsv1.Deployment, resources FunctionResources) string {
	// execute a "dry run" to update the local "deploy" variable with all
	// mutated configurations
	resources.Update(context.TODO(), dep, client.DryRunAll)
	return CalculateHash(dep.Spec)
}

// pulpcoreEnvVars retuns the list of variable names that are defined by pulp-operator
func pulpcoreEnvVars() map[string]struct{} {
	envVarNames := []string{
		"PULP_GUNICORN_TIMEOUT", "PULP_API_WORKERS",
		"PULP_CONTENT_WORKERS", "REDIS_SERVICE_HOST",
		"REDIS_SERVICE_PORT", "REDIS_SERVICE_DB",
		"REDIS_SERVICE_PASSWORD", "PULP_SIGNING_KEY_FINGERPRINT",
		"POSTGRES_SERVICE_HOST", "POSTGRES_SERVICE_PORT",
	}

	envVars := map[string]struct{}{}
	for _, v := range envVarNames {
		envVars[v] = struct{}{}
	}

	return envVars
}

// isPulpcoreEnvVar return true if envVar is in the list of variables
// managed/defined by pulp-operator
func isPulpcoreEnvVar(envVar string) bool {
	envVars := pulpcoreEnvVars()
	if _, found := envVars[envVar]; found {
		return true
	}
	return false
}

// setCustomEnvVars returns the list of custom environment variables defined in Pulp CR
func SetCustomEnvVars(pulp repomanagerpulpprojectorgv1beta2.Pulp, component string) []corev1.EnvVar {
	userDefinedVars := []corev1.EnvVar{}

	switch component {
	case string(settings.API), string(settings.WORKER), string(settings.CONTENT):
		userDefinedVars = append(userDefinedVars, reflect.ValueOf(pulp.Spec).FieldByName(component).FieldByName("EnvVars").Interface().([]corev1.EnvVar)...)
	default: // if it is not a pulpcore component it is a job
		userDefinedVars = append(userDefinedVars, reflect.ValueOf(pulp.Spec).FieldByName(component).FieldByName("PulpContainer").FieldByName("EnvVars").Interface().([]corev1.EnvVar)...)
	}

	envVars := []corev1.EnvVar{}
	for _, v := range userDefinedVars {
		if isPulpcoreEnvVar(v.Name) {
			CustomZapLogger().Warn("The " + v.Name + " env var is managed by pulp-operator and will be ignored!")
			continue
		}
		envVars = append(envVars, v)
	}
	return envVars
}

// setPulpcoreCustomEnvVars returns the list of custom environment variables defined in Pulp CR
func SetPulpcoreCustomEnvVars(pulp repomanagerpulpprojectorgv1beta2.Pulp, pulpcoreType settings.PulpcoreType) []corev1.EnvVar {
	return SetCustomEnvVars(pulp, string(pulpcoreType))
}

// GetStorageType retrieves the storage type defined in pulp CR
func GetStorageType(pulp repomanagerpulpprojectorgv1beta2.Pulp) []string {
	_, storageType := MultiStorageConfigured(&pulp, "Pulp")
	return storageType
}

// DeployCollectionSign returns true if signingScript secret is defined with a collection script
func DeployCollectionSign(secret corev1.Secret) bool {
	_, contains := secret.Data[settings.CollectionSigningScriptName]
	return contains
}

// DeployContainerSign returns true if signingScript secret is defined with a container script
func DeployContainerSign(secret corev1.Secret) bool {
	_, contains := secret.Data[settings.ContainerSigningScriptName]
	return contains
}

// SetDefaultSecurityContext defines the container security configuration to be in compliance with PodSecurity "restricted:v1.24"
func SetDefaultSecurityContext() *corev1.SecurityContext {
	allowPrivilegeEscalation, runAsNonRoot := false, true
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: &runAsNonRoot,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}
