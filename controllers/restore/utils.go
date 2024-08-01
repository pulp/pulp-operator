package repo_manager_restore

import (
	"context"
	"time"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const CMLock = "restore-lock"

// isFileStorage returns true if pulp is deployed with storage type = file
// this is a workaround to identify if it will be necessary to mount /var/lib/pulp in the backup-manager pod
// to restore its contents
func (r *RepoManagerRestoreReconciler) isFileStorage(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) bool {
	// if file-storage PVC is not provisioned it means that pulp is deployed with object storage
	// in this case, we should just return restorePulpDir without action
	fileStoragePVC := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.DeploymentName + "-file-storage", Namespace: pulpRestore.Namespace}, fileStoragePVC); err != nil {
		return false
	}

	return true
}

// backupPVCFound returns the name of PVC and true if backup-claim PVC is found else return nil,false
func (r *RepoManagerRestoreReconciler) backupPVCFound(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) (string, bool) {

	backupPVCName := ""
	if pulpRestore.Spec.BackupPVC == "" {
		backupPVCName = pulpRestore.Spec.BackupName + "-backup-claim"
	} else {
		backupPVCName = pulpRestore.Spec.BackupPVC
	}
	backupPVC := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: backupPVCName, Namespace: pulpRestore.Namespace}, backupPVC); err != nil {
		return "", false
	}
	return backupPVCName, true

}

// [TODO] refactor updateStatus so that it can be used by pulp, pulpRestore, and pulpBackup controllers
// updateStatus modifies a .status.condition from pulpbackup CR
func (r *RepoManagerRestoreReconciler) updateStatus(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, conditionStatus metav1.ConditionStatus, conditionType, conditionMessage, conditionReason string) {
	v1.SetStatusCondition(&pulpRestore.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             conditionReason,
		LastTransitionTime: metav1.Now(),
		Message:            conditionMessage,
	})
	r.Status().Update(ctx, pulpRestore)
}

// [TODO] refactor cleanup so that it can be used by pulpRestore and pulpBackup controllers
// cleanup deletes the backup-manager pod
func (r *RepoManagerRestoreReconciler) cleanup(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) error {
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

// [TODO] refactor createBackupPod so that it can be used by pulpRestore and pulpBackup controllers
// createBackupPod provisions the backup-manager pod where the restore steps will run
func (r *RepoManagerRestoreReconciler) createRestorePod(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, backupPVCName, backupDir string) (*corev1.Pod, error) {
	log := r.RawLogger

	labels := map[string]string{
		"app.kubernetes.io/name":       pulpRestore.Spec.DeploymentType + "-backup-storage",
		"app.kubernetes.io/instance":   pulpRestore.Spec.DeploymentType + "-backup-storage-" + pulpRestore.Name,
		"app.kubernetes.io/component":  "backup-storage",
		"app.kubernetes.io/part-of":    pulpRestore.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulpRestore.Spec.DeploymentType + "-operator",
	}

	// [TO-DO] define postgres image based on the database implementation type
	// if external database: we should gather from an user input (pulpRestore CR) postgres version
	// if provisioned by operator: we should gather, for example, from pulp CR spec or from database deployment spec
	postgresImage := "docker.io/library/postgres:13"

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

	// we will only mount file-storage PVC if it is found
	if r.isFileStorage(ctx, pulpRestore) {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		})

		volumes = append(volumes, corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pulpRestore.Spec.DeploymentName + "-file-storage",
				},
			},
		})
	}

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

	runAsUser := int64(700)
	fsGroup := int64(700)

	restorePod := &corev1.Pod{}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpRestore.Name + "-backup-manager",
			Namespace: pulpRestore.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            pulpRestore.Name + "-backup-manager",
				Image:           postgresImage,
				ImagePullPolicy: corev1.PullAlways,
				Command: []string{
					"sleep",
					"infinity",
				},
				VolumeMounts:    volumeMounts,
				ReadinessProbe:  readinessProbe,
				SecurityContext: controllers.SetDefaultSecurityContext(),
			}},
			Volumes:         volumes,
			RestartPolicy:   corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup},
		},
	}
	err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Name + "-backup-manager", Namespace: pulpRestore.Namespace}, restorePod)
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

// waitPodReady waits until container gets into a "READY" state or 120 seconds timeout
func (r *RepoManagerRestoreReconciler) waitPodReady(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
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

// createLockConfigMap creates a new configmap that is used to control the operator execution.
// If this configmap is present it means that a restore has been done and the restore-controller
// should not try a new execution.
// This is to avoid scenarios in which a restore CR is kept after a restore already finished and
// a new reconciliation loop run trying to overwrite the resources when it shouldn't.
// To rerun a restore the user will have to manually delete the lock configmap first.
func (r *RepoManagerRestoreReconciler) createLockConfigMap(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) {
	log := r.RawLogger

	configMap := &corev1.ConfigMap{}
	r.Get(ctx, types.NamespacedName{Name: CMLock, Namespace: pulpRestore.Namespace}, configMap)

	currentTime := time.Now()
	lockCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CMLock,
			Namespace: pulpRestore.Namespace,
			Labels: map[string]string{
				"owner":   "pulp-restore",
				"app":     "pulp-restore",
				"pulp_cr": pulpRestore.Name,
			},
		},
		Data: map[string]string{
			"last-restore-execution": currentTime.Format("2006-01-02 15:04:05"),
		},
	}

	ctrl.SetControllerReference(pulpRestore, lockCM, r.Scheme)

	// create the configmap
	if err := r.Create(ctx, lockCM); err != nil {
		log.Error(err, "Failed to create the PulpRestore Lock ConfigMap")
	}
}

// getBackupDir return the name of backup folder
// if pulpRestore.Spec.BackupDir is not defined it will get the name from pulpBackup status
// if pulpRestore.Spec.BackupDir is not defined and pulpBackup is not found it will return error
func (r *RepoManagerRestoreReconciler) getBackupDir(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) (string, error) {
	log := r.RawLogger

	backupDir := pulpRestore.Spec.BackupDir
	if len(pulpRestore.Spec.BackupDir) == 0 {
		pulpBackup := &repomanagerpulpprojectorgv1beta2.PulpBackup{}
		if err := r.Get(ctx, types.NamespacedName{Name: pulpRestore.Spec.BackupName, Namespace: pulpRestore.Namespace}, pulpBackup); err != nil {
			log.Error(err, "Failed to get pulpBackup and no backup_dir provided!")
			return "", err
		}
		return pulpBackup.Status.BackupDirectory, nil
	}
	return backupDir, nil
}

// getDeploymentName returns the deployment_name
func getDeploymentName(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) string {
	return pulpRestore.Spec.DeploymentName
}
