package controllers

import (
	"context"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// backupPulpDir copies the content of /var/lib/pulp into the backup PVC
func (r *PulpBackupReconciler) backupPulpDir(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	log.Info("Starting pulp dir backup ...")
	execCmd := []string{
		"mkdir", "-p", backupDir + "/pulp",
	}
	_, err := r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to create pulp backup dir")
		return err
	}

	execCmd = []string{
		"bash", "-c", "cp -fr /var/lib/pulp/. " + backupDir + "/pulp",
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup pulp dir")
		return err
	}

	log.Info(pulpBackup.Spec.DeploymentType + "'s directory backup finished!")
	return nil
}
