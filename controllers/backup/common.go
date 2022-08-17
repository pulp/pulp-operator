package pulp_backup

import (
	"bytes"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
)

// containerExec runs []command in the container
func (r *PulpBackupReconciler) containerExec(pod *corev1.Pod, command []string, container, namespace string) (string, error) {
	execReq := r.RESTClient.
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
		}, runtime.NewParameterCodec(r.Scheme))

	exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", execReq.URL())
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

	// I think the exec.Stream command is not synchronous and sometimes when a task depends
	// on the results of the previous one it is failing.
	// But this is just a guess!!! We need to investigate it further.
	time.Sleep(time.Second)
	return result, nil
}
