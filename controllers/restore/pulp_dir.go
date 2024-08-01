package repo_manager_restore

import (
	"context"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
)

// backupPulpDir copies the content of /var/lib/pulp into the backup PVC
func (r *RepoManagerRestoreReconciler) restorePulpDir(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, backupPVCName, backupDir string) error {

	// if file-storage PVC is not provisioned it means that pulp is deployed with object storage
	// in this case, we should just return without action
	if !r.isFileStorage(ctx, pulpRestore) {
		return nil
	}

	log := r.RawLogger

	// redeploy manager pod to remount the file-storage pvc which
	// has been reprovisioned after restoring pulp CR
	r.cleanup(ctx, pulpRestore)
	pod, err := r.createRestorePod(ctx, pulpRestore, backupPVCName, "/backups")
	if err != nil {
		return err
	}
	log.Info("Starting pulp dir restore ...")
	execCmd := []string{
		"bash", "-c", "cp -fa " + backupDir + "/pulp/ /var/lib/pulp",
	}
	if _, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace); err != nil {
		log.Error(err, "Failed to restore pulp dir")
		return err
	}

	log.Info(pulpRestore.Spec.DeploymentType + "'s directory backup finished!")

	return nil
}
