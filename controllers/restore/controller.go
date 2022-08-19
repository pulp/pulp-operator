/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pulp_restore

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/git-hyagi/pulp-operator-go/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PulpRestoreReconciler reconciles a PulpRestore object
type PulpRestoreReconciler struct {
	client.Client
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulprestores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulprestores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulprestores/finalizers,verbs=update
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulpbackup;pulp,verbs=get;list;
func (r *PulpRestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	backupDir := "/backup"

	pulpRestore := &repomanagerv1alpha1.PulpRestore{}
	err := r.Get(ctx, req.NamespacedName, pulpRestore)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("PulpRestore resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PulpRestore")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restore process running ...", "StartingRestoreProcess")

	// [TODO] REVIEW THIS
	// I'm not sure if we should keep this approach
	// in a DR scenario where ocp cluster is lost but the storage data is safe we should be able to recover
	// pulp without a running pulpBackup
	// Another scenario is recovering from a Pulp project/namespace removal without losing backup PV
	// we should be able to recover only with the data from pulpRestore CR + backup PVC
	// the problem with this approach is that users will need to know (or manually retrieve from the backup) the name of the bkp secrets,
	// the name of the files, and manually configure pulpRestore CR with them
	/* 	// Look up details for the backup
	   	pulpBackup := &repomanagerv1alpha1.PulpBackup{}
	   	err = r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.BackupName, Namespace: pulpRestore.Namespace}, pulpBackup)

	   	// Surface error to user
	   	if err != nil && errors.IsNotFound(err) {
	   		log.Error(err, "PulpBackup CR not found!")
	   		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "PulpBackup CR not found!", "FailedToFindPulpBackupCR")
	   		return ctrl.Result{}, err
	   	}

	   	// make sure that there is a bkpPVC and pulpBackup ran before continue
		// checking only for bkpPVC can get into a state where pulpBackup controller is running and the content from bkpPVC is outdated
		// checking only for pulpBackup status can get into a situation where it ran before but the pvc was removed
	   	for timeout := 0; timeout < 300; timeout++ {
	   		if ###CHECK BKPPVC ### && v1.IsStatusConditionTrue(pulpBackup.Status.Conditions, "BackupComplete") {
	   			break
	   		}
	   		time.Sleep(time.Second * 5)
	   		r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.BackupName, Namespace: pulpRestore.Namespace}, pulpBackup)
	   	} */

	// Fail early if pvc is defined but does not exist
	backupPVCName, PVCfound := r.backupPVCFound(ctx, pulpRestore)
	if !PVCfound {
		log.Error(err, "PVC "+backupPVCName+" not found!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "PVC "+backupPVCName+" not found!", "BackupPVCNotFound")
		return ctrl.Result{}, err
	}

	// Delete any existing management pod
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Removing old manager pod ...", "RemovingOldPod")
	r.cleanup(ctx, pulpRestore)

	// Create a new management pod
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Creating manager pod ...", "CreatingPod")
	pod, err := r.createRestorePod(ctx, pulpRestore, backupPVCName, backupDir)
	if err != nil {
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to create manager pod!", "FailedCreatingPod")
		return ctrl.Result{}, err
	}

	// Check to make sure backup directory exists on PVC
	execCmd := []string{
		"stat", backupDir,
	}
	_, err = r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		// requeue request when backupDir is not found
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to find "+backupDir+" dir!", "BackupDirNotFound")
		return ctrl.Result{}, err
	}

	// Restoring the secrets
	if err := r.restoreSecret(ctx, pulpRestore, backupDir, pod); err != nil {
		// requeue request when there is an error with a secret restore
		return ctrl.Result{}, err
	}

	// Restoring pulp CR
	if err := r.restorePulpCR(ctx, pulpRestore, backupDir, pod); err != nil {
		// requeue request when there is an error with a pulp CR restore
		return ctrl.Result{}, err
	}

	// Restoring database
	if err := r.restoreDatabaseData(ctx, pulpRestore, backupDir, pod); err != nil {
		// requeue request when there is an error with a database restore
		return ctrl.Result{}, err
	}

	// Restoring /var/lib/pulp data
	if err := r.restorePulpDir(ctx, pulpRestore, backupPVCName, backupDir, pod); err != nil {
		// requeue request when there is an error with pulp dir restore
		return ctrl.Result{}, err
	}

	// reprovision pods
	pulp := &repomanagerv1alpha1.Pulp{}
	r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName, Namespace: pulpRestore.Namespace}, pulp)
	pulp.Spec.Api.Replicas = 1
	pulp.Spec.Content.Replicas = 2
	pulp.Spec.Worker.Replicas = 2
	pulp.Spec.Web.Replicas = 1
	if err := r.Update(ctx, pulp); err != nil {
		log.Error(err, "Failed to scale up deployment replicas!")
		return ctrl.Result{}, err
	}

	log.Info("Cleaning up restore resources ...")
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Cleaning up restore resources ...", "DeletingMgmtPod")

	r.cleanup(ctx, pulpRestore)
	r.updateStatus(ctx, pulpRestore, metav1.ConditionTrue, "RestoreComplete", "All restore tasks run!", "RestoreTasksFinished")
	log.Info("Restore tasks finished!")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PulpRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.PulpRestore{}).
		WithEventFilter(controllers.IgnoreUpdateCRStatusPredicate()).
		Complete(r)
}
