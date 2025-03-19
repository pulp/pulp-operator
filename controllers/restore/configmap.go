package repo_manager_restore

import (
	"context"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

// restoreConfigMap restores the operator secrets created by pulpbackup CR
func (r *RepoManagerRestoreReconciler) restoreConfigMap(ctx context.Context, pulpRestore *pulpv1.PulpRestore, backupDir string, pod *corev1.Pod) error {

	r.RawLogger.V(1).Info("Restoring from golang backup version")

	// restore pulp_custom_settings configmap
	if _, err := r.restoreConfigMapFromYaml(ctx, "CustomPulpSettings", backupDir, "custom_pulp_settings.yaml", pod, pulpRestore); err != nil {
		return err
	}

	return nil
}

// restoreConfigMapFromYaml restores the Secret from a YAML file.
func (r *RepoManagerRestoreReconciler) restoreConfigMapFromYaml(ctx context.Context, resourceType, backupDir, backupFile string, pod *corev1.Pod, pulpRestore *pulpv1.PulpRestore) (bool, error) {

	log := r.RawLogger
	configMapFile := backupDir + "/" + backupFile
	execCmd := []string{
		"test", "-f", configMapFile,
	}
	_, err := controllers.ContainerExec(ctx, r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)

	// if configmap is not found there is nothing to be restored
	if err != nil {
		return false, nil
	}
	execCmd = []string{
		"cat", configMapFile,
	}
	cmdOutput, err := controllers.ContainerExec(ctx, r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to get "+backupFile+"!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get "+backupFile, "FailedGet"+resourceType+"ConfigMap")
		return true, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(cmdOutput), nil, nil)
	if err != nil {
		log.Error(err, "Failed to decode ConfigMap!")
	}
	cm := obj.(*corev1.ConfigMap)

	// "removing" fields from backup to avoid errors
	cm.ObjectMeta.ResourceVersion = ""
	cm.ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}

	// we'll recreate the configmap only if it was not found
	// in situations like during a pulpRestore reconcile loop (because of an error) the configmap could have been previously created
	// this will avoid an infinite reconciliation loop trying to recreate a resource that already exists
	if err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: pulpRestore.Namespace}, cm); err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, cm); err != nil {
			log.Error(err, "Failed to create "+resourceType+" configmap!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Error trying to restore "+resourceType+" configmap!", "FailedCreate"+resourceType+"ConfigMap")
			return true, err
		}
		log.Info(resourceType + " ConfigMap restored")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", resourceType+" configmap restored", resourceType+"ConfigMapRestored")
	}

	return true, nil
}
