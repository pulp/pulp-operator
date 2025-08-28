package repo_manager_backup

import (
	"context"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// backupPulpDir copies the content of /var/lib/pulp into the backup PVC
func (r *RepoManagerBackupReconciler) backupPulpDir(ctx context.Context, pulpBackup *pulpv1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	deploymentName := getDeploymentName(pulpBackup)
	backupPod := pulpBackup.Name + "-backup-manager"

	pulp := &pulpv1.Pulp{}
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp); err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	if len(pulp.Spec.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) == 0 && len(pulp.Spec.ObjectStorageGCSSecret) == 0 {
		log.Info("Starting pulp dir backup ...")
		execCmd := []string{
			"mkdir", "-p", backupDir + "/pulp",
		}
		_, err := controllers.ContainerExec(ctx, r, pod, execCmd, backupPod, pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to create pulp backup dir")
			return err
		}

		execCmd = []string{
			"bash", "-c", "cp -fa /var/lib/pulp/. " + backupDir + "/pulp",
		}
		_, err = controllers.ContainerExec(ctx, r, pod, execCmd, backupPod, pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to backup pulp dir")
			return err
		}
		log.Info("Pulp's directory backup finished!")
	}

	return nil
}
