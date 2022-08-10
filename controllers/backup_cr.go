package controllers

import (
	"context"
	"encoding/json"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *PulpBackupReconciler) backupCR(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName, Namespace: pulpBackup.Namespace}, pulp)

	// CR BACKUP
	log.Info("Starting pulp CR backup process ...")
	pulpSpec, _ := json.Marshal(pulp.Spec)
	execCmd := []string{
		"bash", "-c", "echo '" + string(pulpSpec) + "' > " + backupDir + "/cr_object",
	}
	_, err := r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup pulp CR")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
