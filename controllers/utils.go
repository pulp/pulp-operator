package controllers

import (
	"bytes"
	"context"
	"reflect"
	"regexp"
	"strings"
	"time"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
)

// ignoreUpdateCRStatusPredicate filters update events on pulpbackup CR status
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

	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// multiStorageConfigured returns true if Pulp CR is configured with more than one "storage type"
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
	err = exec.Stream(remotecommand.StreamOptions{
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

// isNginxIngressSupported returns true if the provided class has nginx as controller
func IsNginxIngressSupported[T any](resource T, ingressClassName string) bool {
	// get the concrete value of client ({PulpBackup,RepoManagerBackupReconciler,RepoManagerRestoreReconciler})
	clientConcrete := reflect.ValueOf(resource)
	restClient := reflect.Indirect(clientConcrete).FieldByName("Client").Elem().Interface().(client.Client)

	ic := &netv1.IngressClass{}
	if err := restClient.Get(context.TODO(), types.NamespacedName{Name: ingressClassName}, ic); err == nil {
		return ic.Spec.Controller == "k8s.io/ingress-nginx"
	}
	return false
}

// customZapLogger should be used only for warn messages
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
