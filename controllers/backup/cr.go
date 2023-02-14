package repo_manager_backup

import (
	"context"
	"encoding/json"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *RepoManagerBackupReconciler) backupCR(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	deploymentName := getDeploymentName(ctx, pulpBackup)
	deploymentType := getDeploymentType(ctx, pulpBackup)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerpulpprojectorgv1beta2.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	// CR BACKUP
	log.Info("Starting " + deploymentType + " CR backup process ...")
	pulpSpec, _ := json.Marshal(pulp.Spec)
	execCmd := []string{
		"bash", "-c", "echo '" + string(pulpSpec) + "' > " + backupDir + "/cr_object",
	}
	_, err = controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+deploymentType+" CR")
		return err
	}
	return nil
}
