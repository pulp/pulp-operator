package repo_manager_backup

import (
	"context"

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// backupPulpDir copies the content of /var/lib/pulp into the backup PVC
func (r *RepoManagerBackupReconciler) backupPulpDir(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger

	pulp := &repomanagerv1alpha1.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.DeploymentName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	if len(pulp.Spec.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) == 0 {
		log.Info("Starting pulp dir backup ...")
		execCmd := []string{
			"mkdir", "-p", backupDir + "/pulp",
		}
		_, err := controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to create pulp backup dir")
			return err
		}

		execCmd = []string{
			"bash", "-c", "cp -fa /var/lib/pulp/. " + backupDir + "/pulp",
		}
		_, err = controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to backup pulp dir")
			return err
		}
		log.Info(pulpBackup.Spec.DeploymentType + "'s directory backup finished!")
	}

	return nil
}
