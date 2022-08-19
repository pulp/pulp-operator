package pulp_restore

import (
	"context"
	"encoding/json"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// restorePulpCR recreates the pulp CR with the content from backup
func (r *PulpRestoreReconciler) restorePulpCR(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {
	pulp := &repomanagerv1alpha1.Pulp{}

	// we'll recreate pulp instance only if it was not found
	// in situations like during a pulpRestore reconcile loop (because of an error, for example) pulp instance could have been previously created
	// this will avoid an infinite reconciliation loop trying to recreate a resource that already exists
	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp); err != nil && errors.IsNotFound(err) {
		log := ctrllog.FromContext(ctx)
		log.Info("Restoring " + pulpRestore.Spec.DeploymentName + " CR ...")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restoring "+pulpRestore.Spec.DeploymentName+" CR", "Restoring"+pulpRestore.Spec.DeploymentName+"CR")
		execCmd := []string{
			"cat", backupDir + "/cr_object",
		}
		cmdOutput, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to get cr_object backup file!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get cr_object backup file!", "FailedGet"+pulpRestore.Spec.DeploymentName+"CR")
			return err
		}

		pulp := repomanagerv1alpha1.Pulp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pulpRestore.Spec.DeploymentName,
				Namespace: pulpRestore.Namespace,
			},
		}

		json.Unmarshal([]byte(cmdOutput), &pulp.Spec)
		if err = r.Create(ctx, &pulp); err != nil {
			log.Error(err, "Error trying to restore "+pulpRestore.Spec.DeploymentName+" CR!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to restore cr_object!", "FailedRestore"+pulpRestore.Spec.DeploymentName+"CR")
			return err
		}

		log.Info(pulpRestore.Spec.DeploymentName + " CR restored!")
	}

	return nil
}
