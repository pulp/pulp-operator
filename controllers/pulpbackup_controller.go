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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
)

// PulpBackupReconciler reconciles a PulpBackup object
type PulpBackupReconciler struct {
	client.Client
	RESTClient rest.Interface
	RESTConfig *rest.Config
	Scheme     *runtime.Scheme
}

//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulpbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulpbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulpbackups/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods;persistentvolumes;persistentvolumeclaims,verbs=create;update;patch;delete;watch;get;list;
//+kubebuilder:rbac:groups=repo-manager.pulpproject.org,resources=pulp,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// [TO-DO]
// - should we follow the same approach of deleting the backup manager pod if it is available
//   as a first step?
func (r *PulpBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	pulpBackup := &repomanagerv1alpha1.PulpBackup{}
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
		"app.kubernetes.io/name":       pulpBackup.Spec.DeploymentType + "-backup-storage",
		"app.kubernetes.io/instance":   pulpBackup.Spec.DeploymentType + "-backup-storage-" + pulpBackup.Name,
		"app.kubernetes.io/component":  "backup-storage",
		"app.kubernetes.io/part-of":    pulpBackup.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": pulpBackup.Spec.DeploymentType + "-operator",
	}

	// create backup pvc
	pvcFound := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Name + "-backup-claim", Namespace: pulpBackup.Namespace}, pvcFound)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpBackup.Name + "-backup-claim",
			Namespace: pulpBackup.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
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
			return ctrl.Result{}, err
		}
		// PVC created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get PulpBackup PVC")
		return ctrl.Result{}, err
	}

	// create management pod
	labels = map[string]string{
		"app.kubernetes.io/name":      pulpBackup.Spec.DeploymentType + "-backup-manager",
		"app.kubernetes.io/instance":  pulpBackup.Spec.DeploymentType + "-backup-manager-" + pulpBackup.Name,
		"app.kubernetes.io/component": "backup-manager",
	}

	// [TO-DO] define postgres image based on the database implementation type
	// if external database: we should gather from an user input (pulpbackup CR) postgres version
	// if provisioned by operator: we should gather, for example, from pulp CR spec or from database deployment spec
	postgresImage := "postgres:13"
	volumeMounts := []corev1.VolumeMount{{
		Name:      pulpBackup.Name + "-backup",
		ReadOnly:  false,
		MountPath: "/backups",
	}}

	volumes := []corev1.Volume{{
		Name: pulpBackup.Name + "-backup",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pulpBackup.Name + "-backup-claim",
			},
		},
	}}

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	r.Get(ctx, types.NamespacedName{Namespace: pulpBackup.Namespace}, pulp)

	if pulp.Spec.IsFileStorage {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		})

		volumes = append(volumes, corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pulpBackup.Name + "-file-storage",
				},
			},
		})
	}

	podFound := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Name + "-backup-manager", Namespace: pulpBackup.Namespace}, podFound)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulpBackup.Name + "-backup-manager",
			Namespace: pulpBackup.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            pulpBackup.Name + "-backup-manager",
				Image:           postgresImage,
				ImagePullPolicy: corev1.PullAlways,
				Command: []string{
					"sleep",
					"infinity",
				},
				VolumeMounts: volumeMounts,
			}},
			Volumes:       volumes,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new backup manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		ctrl.SetControllerReference(pulpBackup, pod, r.Scheme)
		err = r.Create(ctx, pod)
		if err != nil {
			log.Error(err, "Failed to create new backup manager Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			return ctrl.Result{}, err
		}
		// Pod created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get backup manager Pod")
		return ctrl.Result{}, err
	}

	backupFile := "/tmp/pulp.db"

	log.Info("Backup pod running")
	execCmd := []string{"touch", backupFile}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to execute command inside container")
		return ctrl.Result{}, err
	}

	execCmd = []string{"chmod", "0600", backupFile}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to execute command inside container")
		return ctrl.Result{}, err
	}

	postgresHost := "postgres.db.svc.cluster.local"
	postgresUser := "pulp-admin"
	postgresDB := "pulp"
	postgresPort := "5432"
	postgresPwd := "password"
	execCmd = []string{
		"pg_dump", "--clean", "--create",
		"-d", "postgresql://" + postgresUser + ":" + postgresPwd + "@" + postgresHost + ":" + postgresPort + "/" + postgresDB,
		"-f", backupFile,
	}

	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to execute command inside container")
		return ctrl.Result{}, err
	}

	log.Info("DONE!")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PulpBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&repomanagerv1alpha1.PulpBackup{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
