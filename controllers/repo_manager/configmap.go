package repo_manager

import (
	"context"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) configMapTasks(ctx context.Context, pulp *pulpv1.Pulp) (*ctrl.Result, error) {

	needsPulpcoreRestart := false

	// if mount_trusted_ca_configmap_key is defined, check if it was modified
	if pulp.Spec.TrustedCaConfigMapKey != nil {
		trustedCAConfigMap := &corev1.ConfigMap{}
		caConfigMapName, _ := controllers.SplitCAConfigMapNameKey(*pulp)

		r.Get(ctx, types.NamespacedName{Name: caConfigMapName, Namespace: pulp.Namespace}, trustedCAConfigMap)
		if r.caConfigMapChanged(ctx, trustedCAConfigMap) {
			needsPulpcoreRestart = true
		}
	}

	// TODO: check pulp-web configmap change

	// restart pulpcore pods if any of the configmaps changed
	if needsPulpcoreRestart {
		r.restartPulpCorePods(ctx, pulp)
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

func (r *RepoManagerReconciler) caConfigMapChanged(ctx context.Context, cm *corev1.ConfigMap) bool {
	currentHash := controllers.GetCurrentHash(cm)
	calculatedHash := controllers.CalculateHash(cm.Data)

	if currentHash == calculatedHash {
		return false
	}

	controllers.SetHashLabel(calculatedHash, cm)
	if err := r.Update(ctx, cm); err != nil {
		r.RawLogger.Error(err, "Failed to update "+cm.Name+" ConfigMap label!")
	}
	return true
}
