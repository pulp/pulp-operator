package controllers

import (
	"context"
	"fmt"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"gopkg.in/yaml.v3"
)

type adminPassword struct {
	Admin_password_secret string
	Password              string
}

func (r *PulpRestoreReconciler) restoreSecret(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	// restore admin password secret
	execCmd := []string{
		"cat", backupDir + "/admin_secret.yaml",
	}
	cmdOutput, err := containerExec(pod, r, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to find "+backupDir+" dir!", "BackupDirNotFound")
		return err
	}

	log.Info(cmdOutput)
	t := adminPassword{}
	yaml.Unmarshal([]byte(cmdOutput), &t)

	fmt.Println(t)

	return nil
}
