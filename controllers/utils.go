package controllers

import (
	"bytes"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

// ContainerExec runs a command in the container
func ContainerExec[T any](client T, pod *corev1.Pod, command []string, container, namespace string) (string, error) {

	// get the concrete value of client ({PulpBackup,PulpBackupReconciler,PulpRestoreReconciler})
	clientConcrete := reflect.ValueOf(client)

	// here we are using the Indirect method to get the value where client is pointing to
	// after that we are taking the RESTClient field from PulpBackup|PulpBackupReconciler|PulpRestoreReconciler and
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
