package pulp_restore

import (
	"context"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// backupPulpDir copies the content of /var/lib/pulp into the backup PVC
func (r *PulpRestoreReconciler) restorePulpDir(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupPVCName, backupDir string, pod *corev1.Pod) error {

	// if file-storage PVC is not provisioned it means that pulp is deployed with object storage
	// in this case, we should just return without action
	if !r.isFileStorage(ctx, pulpRestore) {
		return nil
	}

	log := ctrllog.FromContext(ctx)

	// redeploy manager pod to remount the file-storage pvc which
	// has been reprovisioned after restoring pulp CR
	r.cleanup(ctx, pulpRestore)
	r.createRestorePod(ctx, pulpRestore, backupPVCName, backupDir)

	log.Info("Starting pulp dir restore ...")
	execCmd := []string{
		"bash", "-c", "cp -fr " + backupDir + "/pulp/. /var/lib/pulp",
	}
	if _, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace); err != nil {
		log.Error(err, "Failed to restore pulp dir")
		return err
	}

	log.Info(pulpRestore.Spec.DeploymentType + "'s directory backup finished!")

	return nil
}
