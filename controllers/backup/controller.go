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

package repo_manager_backup

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RepoManagerBackupReconciler reconciles a PulpBackup object
type RepoManagerBackupReconciler struct {
	client.Client
	RawLogger  logr.Logger
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulpbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulpbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulpbackups/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=pods;persistentvolumes;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list;
//+kubebuilder:rbac:groups=core,namespace=pulp-operator-system,resources=pods/exec,verbs=create;
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,namespace=pulp-operator-system,resources=pulps,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RepoManagerBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.RawLogger

	formattedCurrentTime := time.Now().Format("2006-01-02-150405")
	pulpBackup := &repomanagerpulpprojectorgv1beta2.PulpBackup{}
	err := r.Get(ctx, req.NamespacedName, pulpBackup)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("PulpBackup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PulpBackup")
		return ctrl.Result{}, err
	}
	backupDir := getBackupDir(ctx, pulpBackup, formattedCurrentTime)
	deploymentType := getDeploymentType(ctx, pulpBackup)

	if err := checkRequiredFields(ctx, pulpBackup); err != nil {
		log.Error(err, "Required field not filled in backup CR!")
		return ctrl.Result{}, nil
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Backup process running ...", "StartingBackupProcess")
	r.cleanup(ctx, pulpBackup)

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Creating backup pvc ...", "CreatingPVC")
	err = r.createBackupPVC(ctx, pulpBackup)
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to create backup pvc!", "FailedCreatingPVC")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Creating backup pod ...", "CreatingPod")
	pod, err := r.createBackupPod(ctx, pulpBackup, "/backups")
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to create backup pod!", "FailedCreatingPod")
		return ctrl.Result{}, err
	}

	if err = r.createBackupDir(ctx, pulpBackup, backupDir, pod); err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to create backup directory!", "FailedCreateBkpDir")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Running database backup ...", "BackupDB")
	err = r.backupDatabase(ctx, pulpBackup, backupDir, pod)
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to backup database!", "FailedBackupDB")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Running CR backup ...", "BackupCR")
	err = r.backupCR(ctx, pulpBackup, backupDir, pod)
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to backup CR!", "FailedBackupCR")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Running secrets backup ...", "BackupSecrets")
	err = r.backupSecret(ctx, pulpBackup, backupDir, pod)
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to backup secrets!", "FailedBackupSecrets")
		return ctrl.Result{}, err
	}

	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Running "+deploymentType+" dir backup ...", "BackupDir")
	err = r.backupPulpDir(ctx, pulpBackup, backupDir, pod)
	if err != nil {
		r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Failed to backup "+deploymentType+" dir!", "FailedBackupDir")
		return ctrl.Result{}, err
	}

	log.Info("Cleaning up backup resources ...")
	r.updateStatus(ctx, pulpBackup, metav1.ConditionFalse, "BackupComplete", "Cleaning up backup resources ...", "DeletingBkpPod")
	r.cleanup(ctx, pulpBackup)
	if err := r.setStatusFields(ctx, pulpBackup, formattedCurrentTime); err != nil {
		log.Error(err, "Failed to update backup CR status!")
	}
	r.updateStatus(ctx, pulpBackup, metav1.ConditionTrue, "BackupComplete", "All backup tasks run!", "BackupTasksFinished")
	log.Info(deploymentType + " CR Backup finished!")

	return ctrl.Result{}, nil
}

// waitPodReady waits until container gets into a "READY" state
func (r *RepoManagerBackupReconciler) waitPodReady(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
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

// createBackupPod provisions the backup-manager pod where the backup steps will run
func (r *RepoManagerBackupReconciler) createBackupPod(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, backupDir string) (*corev1.Pod, error) {
	log := r.RawLogger

	deploymentName := getDeploymentName(ctx, pulpBackup)
	deploymentType := getDeploymentType(ctx, pulpBackup)
	backupPVC := getBackupPVC(ctx, pulpBackup)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerpulpprojectorgv1beta2.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get Pulp")
		return nil, err
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       deploymentType + "-backup-storage",
		"app.kubernetes.io/instance":   deploymentType + "-backup-storage-" + pulpBackup.Name,
		"app.kubernetes.io/component":  "backup-storage",
		"app.kubernetes.io/part-of":    deploymentType,
		"app.kubernetes.io/managed-by": deploymentType + "-operator",
	}

	affinity := &corev1.Affinity{}
	if pulpBackup.Spec.Affinity != nil {
		affinity = pulpBackup.Spec.Affinity
	}

	// [TO-DO] define postgres image based on the database implementation type
	// if external database: we should gather from an user input (pulpbackup CR) postgres version
	// if provisioned by operator: we should gather, for example, from pulp CR spec or from database deployment spec
	postgresImage := "docker.io/library/postgres:13"
	volumeMounts := []corev1.VolumeMount{{
		Name:      pulpBackup.Name + "-backup",
		ReadOnly:  false,
		MountPath: backupDir,
	}}

	volumes := []corev1.Volume{{
		Name: pulpBackup.Name + "-backup",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: backupPVC,
			},
		},
	}}

	// fileStorageMount will be added to the list of mounts if there is a
	// SC or PVC defined for Pulp
	fileStorageMount := corev1.VolumeMount{
		Name:      "file-storage",
		ReadOnly:  false,
		MountPath: "/var/lib/pulp",
	}

	// fileStorageVolume will be added to the list of volumes if there is a
	// SC or PVC defined for Pulp
	fileStorageVolume := corev1.Volume{
		Name: "file-storage",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{},
		},
	}

	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")

	// if SC defined, we should mount the PVC provisioned by the operator
	if storageType[0] == controllers.SCNameType {
		volumeMounts = append(volumeMounts, fileStorageMount)
		fileStorageVolume.VolumeSource.PersistentVolumeClaim.ClaimName = pulp.Name + "-file-storage"
		volumes = append(volumes, fileStorageVolume)

		// if .spec.Api.PVC defined we should mount the PVC provisioned by user
	} else if storageType[0] == controllers.PVCType {
		volumeMounts = append(volumeMounts, fileStorageMount)
		fileStorageVolume.VolumeSource.PersistentVolumeClaim.ClaimName = pulp.Spec.PVC
		volumes = append(volumes, fileStorageVolume)
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

	bkpPod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Name + "-backup-manager", Namespace: pulpBackup.Namespace}, bkpPod)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpBackup.Name + "-backup-manager",
			Namespace: pulpBackup.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Affinity: affinity,
			Containers: []corev1.Container{{
				Name:            pulpBackup.Name + "-backup-manager",
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
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new backup manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		ctrl.SetControllerReference(pulpBackup, pod, r.Scheme)
		err = r.Create(ctx, pod)
		if err != nil {
			log.Error(err, "Failed to create new backup manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			return &corev1.Pod{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get backup manager Pod")
		return &corev1.Pod{}, err
	}

	pod, err = r.waitPodReady(ctx, pulpBackup.Namespace, pulpBackup.Name+"-backup-manager")
	if err != nil {
		log.Error(err, "Backup pod not found")
		return &corev1.Pod{}, err
	}
	return pod, nil
}

// cleanup deletes the backup-manager pod
func (r *RepoManagerBackupReconciler) cleanup(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) error {
	bkpPod := &corev1.Pod{}
	r.Get(ctx, types.NamespacedName{Name: pulpBackup.Name + "-backup-manager", Namespace: pulpBackup.Namespace}, bkpPod)
	r.Delete(ctx, bkpPod)

	// the Delete method is not synchronous, so this loop will wait until the pod is not present anymore or
	// the 120 seconds timeout
	for timeout := 0; timeout < 120; timeout++ {
		err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Name + "-backup-manager", Namespace: pulpBackup.Namespace}, bkpPod)
		if err != nil && errors.IsNotFound(err) {
			break
		}
		time.Sleep(time.Second * 5)
	}

	return nil
}

// createBackupPVC provisions the pulp-backup-claim PVC that will store the backup
func (r *RepoManagerBackupReconciler) createBackupPVC(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup) error {
	log := r.RawLogger

	backupPVC := getBackupPVC(ctx, pulpBackup)
	backupPVCNamespace := getBackupPVCNamespace(ctx, pulpBackup)
	deploymentType := getDeploymentType(ctx, pulpBackup)

	var storageClassName string
	if pulpBackup.Spec.BackupSC != "" {
		storageClassName = pulpBackup.Spec.BackupSC
	}

	var storageRequirements string
	if pulpBackup.Spec.BackupStorageReq != "" {
		storageRequirements = pulpBackup.Spec.BackupStorageReq
	} else {
		storageRequirements = "5Gi"
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       deploymentType + "-backup-storage",
		"app.kubernetes.io/instance":   deploymentType + "-backup-storage-" + pulpBackup.Name,
		"app.kubernetes.io/component":  "backup-storage",
		"app.kubernetes.io/part-of":    deploymentType,
		"app.kubernetes.io/managed-by": deploymentType + "-operator",
	}

	// create backup pvc
	pvcFound := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: backupPVC, Namespace: backupPVCNamespace}, pvcFound)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupPVC,
			Namespace: backupPVCNamespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(storageRequirements),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(corev1.ReadWriteOnce),
			},
			StorageClassName: &storageClassName,
		},
	}

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new PulpBackup PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
		err = r.Create(ctx, pvc)
		if err != nil {
			log.Error(err, "Failed to create new PulpBackup PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
			return err
		}
	} else if err != nil {
		log.Error(err, "Failed to get PulpBackup PVC")
		return err
	}

	return nil
}

// updateStatus modifies a .status.condition from pulpbackup CR
func (r *RepoManagerBackupReconciler) updateStatus(ctx context.Context, pulpBackup *repomanagerpulpprojectorgv1beta2.PulpBackup, conditionStatus metav1.ConditionStatus, conditionType, conditionMessage, conditionReason string) {
	v1.SetStatusCondition(&pulpBackup.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             conditionReason,
		LastTransitionTime: metav1.Now(),
		Message:            conditionMessage,
	})
	r.Status().Update(ctx, pulpBackup)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoManagerBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerpulpprojectorgv1beta2.PulpBackup{}).
		WithEventFilter(controllers.IgnoreUpdateCRStatusPredicate()).
		Complete(r)
}
