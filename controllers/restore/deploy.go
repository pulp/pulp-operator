package pulp_restore

import (
	"context"
	"encoding/json"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *PulpRestoreReconciler) restorePulpCR(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	log.Info("Restoring " + pulpRestore.Spec.DeploymentType + " CR ...")
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restoring "+pulpRestore.Spec.DeploymentType+" CR", "Restoring"+pulpRestore.Spec.DeploymentType+"CR")
	execCmd := []string{
		"cat", backupDir + "/cr_object",
	}
	cmdOutput, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to get cr_object backup file!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get cr_object backup file!", "FailedGet"+pulpRestore.Spec.DeploymentType+"CR")
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
		log.Error(err, "Error trying to restore "+pulpRestore.Spec.DeploymentType+" CR!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to restore cr_object!", "FailedRestore"+pulpRestore.Spec.DeploymentType+"CR")
		return err
	}

	return nil
}
