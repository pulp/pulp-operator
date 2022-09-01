package pulp_backup

import (
	"context"
	"encoding/json"

	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *PulpBackupReconciler) backupCR(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.DeploymentName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	// CR BACKUP
	log.Info("Starting " + pulpBackup.Spec.DeploymentType + " CR backup process ...")
	pulpSpec, _ := json.Marshal(pulp.Spec)
	execCmd := []string{
		"bash", "-c", "echo '" + string(pulpSpec) + "' > " + backupDir + "/cr_object",
	}
	_, err = controllers.ContainerExec(r, pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+pulpBackup.Spec.DeploymentType+" CR")
		return err
	}
	return nil
}
