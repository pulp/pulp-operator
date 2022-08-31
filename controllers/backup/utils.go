package pulp_backup

import (
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
)

// containerExec runs []command in the container
func (r *PulpBackupReconciler) containerExec(pod *corev1.Pod, command []string, container, namespace string) (string, error) {
	return controllers.ContainerExec(r.RESTClient, r.Scheme, r.RESTConfig, pod, command, container, namespace)
}
