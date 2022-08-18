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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/git-hyagi/pulp-operator-go/controllers"
	v1 "k8s.io/apimachinery/pkg/api/meta"
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

	/* 	// Look up details for the backup
	   	pulpBackup := &repomanagerv1alpha1.PulpBackup{}
	   	err = r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.BackupName, Namespace: pulpRestore.Namespace}, pulpBackup)

	   	// Surface error to user
	   	if err != nil && errors.IsNotFound(err) {
	   		log.Error(err, "PulpBackup CR not found!")
	   		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "PulpBackup CR not found!", "FailedToFindPulpBackupCR")
	   		return ctrl.Result{}, err
	   	}

	   	// make sure that pulpBackup ran before continue
	   	for timeout := 0; timeout < 300; timeout++ {
	   		if v1.IsStatusConditionTrue(pulpBackup.Status.Conditions, "BackupComplete") {
	   			break
	   		}
	   		time.Sleep(time.Second * 5)
	   		r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.BackupName, Namespace: pulpRestore.Namespace}, pulpBackup)
	   	} */

	// Fail early if pvc is defined but does not exist
	backupPVCName := ""
	if pulpRestore.Spec.BackupPVC == "" {
		//backupPVCName = pulpBackup.Name + "-backup-claim"
		backupPVCName = "pulpbackup-sample-backup-claim"
	} else {
		backupPVCName = pulpRestore.Spec.BackupPVC
	}
	backupPVC := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: backupPVCName, Namespace: pulpRestore.Namespace}, backupPVC)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "PVC "+backupPVCName+" not found!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "PVC "+backupPVCName+" not found!", "PVCNotFound")
		return ctrl.Result{}, err
	}

	// Delete any existing management pod
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Removing old manager pod ...", "RemovingOldPod")
	//r.cleanup(ctx, pulpRestore)

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
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to find "+backupDir+" dir!", "BackupDirNotFound")
		return ctrl.Result{}, err
	}

	r.restoreSecret(ctx, pulpRestore, backupDir, pod)

	r.restorePulpCR(ctx, pulpRestore, backupDir, pod)

	if err = r.restoreDatabaseData(ctx, pulpRestore, backupDir, pod); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Cleaning up restore resources ...")
	r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Cleaning up restore resources ...", "DeletingMgmtPod")
	//r.cleanup(ctx, pulpRestore)
	r.updateStatus(ctx, pulpRestore, metav1.ConditionTrue, "RestoreComplete", "All restore tasks run!", "RestoreTasksFinished")
	log.Info("Restore tasks finished!")
	return ctrl.Result{}, nil
}

// updateStatus modifies a .status.condition from pulpbackup CR
func (r *PulpRestoreReconciler) updateStatus(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, conditionStatus metav1.ConditionStatus, conditionType, conditionMessage, conditionReason string) {
	v1.SetStatusCondition(&pulpRestore.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             conditionReason,
		LastTransitionTime: metav1.Now(),
		Message:            conditionMessage,
	})
	r.Status().Update(ctx, pulpRestore)
}

// cleanup deletes the backup-manager pod
func (r *PulpRestoreReconciler) cleanup(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore) error {
	restorePod := &corev1.Pod{}
	r.Get(ctx, types.NamespacedName{Name: pulpRestore.Name + "-backup-manager", Namespace: pulpRestore.Namespace}, restorePod)
	r.Delete(ctx, restorePod)

	// the Delete method is not synchronous, so this loop will wait until the pod is not present anymore or
	// the 120 seconds timeout
	for timeout := 0; timeout < 120; timeout++ {
		err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Name + "-backup-manager", Namespace: pulpRestore.Namespace}, restorePod)
		if err != nil && errors.IsNotFound(err) {
			break
		}
		time.Sleep(time.Second * 5)
	}

	return nil
}

// createBackupPod provisions the backup-manager pod where the restore steps will run
func (r *PulpRestoreReconciler) createRestorePod(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupPVCName, backupDir string) (*corev1.Pod, error) {
	log := log.FromContext(ctx)

	labels := map[string]string{
		"app.kubernetes.io/name":       pulpRestore.Spec.DeploymentType + "-backup-storage",
		"app.kubernetes.io/instance":   pulpRestore.Spec.DeploymentType + "-backup-storage-" + pulpRestore.Name,
		"app.kubernetes.io/component":  "backup-storage",
		"app.kubernetes.io/part-of":    pulpRestore.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulpRestore.Spec.DeploymentType + "-operator",
	}

	// [TO-DO] define postgres image based on the database implementation type
	// if external database: we should gather from an user input (pulpbackup CR) postgres version
	// if provisioned by operator: we should gather, for example, from pulp CR spec or from database deployment spec
	postgresImage := "postgres:13"

	volumeMounts := []corev1.VolumeMount{{
		Name:      pulpRestore.Name + "-backup",
		ReadOnly:  false,
		MountPath: backupDir,
	}}

	volumes := []corev1.Volume{{
		Name: pulpRestore.Name + "-backup",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: backupPVCName,
			},
		},
	}}

	// running a dumb command on bkp mount point just to make sure that
	// the pod is ready to execute the backup commands (mkdir,cp,echo,etc)
	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{Command: []string{"ls", backupDir}},
		},
		FailureThreshold:    10,
		InitialDelaySeconds: 3,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
	}

	serviceAccount := ""
	if pulpRestore.Spec.DeploymentType == "" {
		serviceAccount = pulpRestore.Spec.DeploymentName + "-operator-sa"
	} else {
		serviceAccount = pulpRestore.Spec.DeploymentType + "-operator-sa"
	}
	restorePod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Name + "-backup-manager", Namespace: pulpRestore.Namespace}, restorePod)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpRestore.Name + "-backup-manager",
			Namespace: pulpRestore.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{{
				Name:            pulpRestore.Name + "-backup-manager",
				Image:           postgresImage,
				ImagePullPolicy: corev1.PullAlways,
				Command: []string{
					"sleep",
					"infinity",
				},
				VolumeMounts:   volumeMounts,
				ReadinessProbe: readinessProbe,
			}},
			Volumes:       volumes,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		ctrl.SetControllerReference(pulpRestore, pod, r.Scheme)
		err = r.Create(ctx, pod)
		if err != nil {
			log.Error(err, "Failed to create new manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			return &corev1.Pod{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get manager Pod")
		return &corev1.Pod{}, err
	}

	pod, err = r.waitPodReady(ctx, pulpRestore.Namespace, pulpRestore.Name+"-backup-manager")
	if err != nil {
		log.Error(err, "Manager pod not found")
		return &corev1.Pod{}, err
	}
	return pod, nil
}

// waitPodReady waits until container gets into a "READY" state
func (r *PulpRestoreReconciler) waitPodReady(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	var err error
	for timeout := 0; timeout < 120; timeout++ {
		pod := &corev1.Pod{}
		err = r.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)

		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return pod, nil
		}
		time.Sleep(time.Second)
	}
	return &corev1.Pod{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PulpRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.PulpRestore{}).
		WithEventFilter(controllers.IgnoreUpdateCRStatusPredicate()).
		Complete(r)
}
