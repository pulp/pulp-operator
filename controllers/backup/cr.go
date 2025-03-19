package repo_manager_backup

import (
	"context"
	"encoding/json"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *RepoManagerBackupReconciler) backupCR(ctx context.Context, pulpBackup *pulpv1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	deploymentName := getDeploymentName(pulpBackup)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &pulpv1.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	// CR BACKUP
	log.Info("Starting Pulp CR backup process ...")
	pulpSpec, _ := json.Marshal(pulp.Spec)
	execCmd := []string{
		"bash", "-c", "echo '" + string(pulpSpec) + "' > " + backupDir + "/cr_object",
	}
	_, err = controllers.ContainerExec(ctx, r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup Pulp CR")
		return err
	}
	return nil
}
