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
	"reflect"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) pulpWorkerController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// Worker Deployment
	workerDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-worker", Namespace: pulp.Namespace}, workerDeployment)
	newWorkerDeployment := r.deploymentForPulpWorker(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Name+"-Worker-Ready", "CreatingWorkerDeployment", "Creating "+pulp.Name+"-worker deployment resource")
		err = r.Create(ctx, newWorkerDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Name+"-Worker-Ready", "ErrorCreatingWorkerDeployment", "Failed to create "+pulp.Name+"-worker deployment resource: "+err.Error())
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Worker Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if !equality.Semantic.DeepDerivative(newWorkerDeployment.Spec, workerDeployment.Spec) {
		log.Info("The Worker Deployment has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Name+"-Worker-Ready", "UpdatingWorkerDeployment", "Reconciling "+pulp.Name+"-worker deployment resource")
		err = r.Update(ctx, newWorkerDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the Worker Deployment object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Name+"-Worker-Ready", "ErrorUpdatingWorkerDeployment", "Failed to reconcile "+pulp.Name+"-worker deployment resource: "+err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Name+"-Worker-Ready", "WorkerTasksFinished", "All Worker tasks ran successfully")
	return ctrl.Result{}, nil
}

// deploymentForPulpWorker returns a pulp-worker Deployment object
func (r *PulpReconciler) deploymentForPulpWorker(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {
	ls := labelsForPulpWorker(m)
	labels := map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-worker",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-worker-" + m.Name,
		"app.kubernetes.io/component":  "worker",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"owner":                        "pulp-dev",
	}
	replicas := m.Spec.Worker.Replicas

	affinity := &corev1.Affinity{}
	if m.Spec.Worker.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Worker.Affinity.NodeAffinity
	}

	runAsUser := int64(0)
	fsGroup := int64(0)
	podSecurityContext := &corev1.PodSecurityContext{}
	if m.Spec.IsK8s {
		podSecurityContext = &corev1.PodSecurityContext{
			RunAsUser: &runAsUser,
			FSGroup:   &fsGroup,
		}
	}

	nodeSelector := map[string]string{}
	if m.Spec.Worker.NodeSelector != nil {
		nodeSelector = m.Spec.Worker.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if m.Spec.Worker.Tolerations != nil {
		toleration = m.Spec.Worker.Tolerations
	}

	dbFieldsEncryptionSecret := ""
	if m.Spec.DBFieldsEncryptionSecret == "" {
		dbFieldsEncryptionSecret = m.Name + "-db-fields-encryption"
	} else {
		dbFieldsEncryptionSecret = m.Spec.DBFieldsEncryptionSecret
	}

	volumes := []corev1.Volume{
		{
			Name: m.Name + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.Name + "-server",
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: m.Name + "-db-fields-encryption",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dbFieldsEncryptionSecret,
					Items: []corev1.KeyToPath{{
						Key:  "database_fields.symmetric.key",
						Path: "database_fields.symmetric.key",
					}},
				},
			},
		},
		{
			Name: m.Name + "-ansible-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	if m.Spec.SigningSecret != "" {
		signingSecretVolume := []corev1.Volume{
			{
				Name: m.Name + "-signing-scripts",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.Spec.SigningScriptsConfigmap,
						},
					},
				},
			},
			{
				Name: m.Name + "-signing-galaxy",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: m.Spec.SigningSecret,
						Items: []corev1.KeyToPath{
							{
								Key:  "signing_service.gpg",
								Path: "signing_serivce.gpg",
							},
							{
								Key:  "signing_service.asc",
								Path: "signing_serivce.asc",
							},
						},
					},
				},
			},
		}
		volumes = append(volumes, signingSecretVolume...)
	}

	if m.Spec.IsFileStorage {
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.Name + "-file-storage",
				},
			},
		}
		volumes = append(volumes, fileStorage)
	} else {
		emptyDir := []corev1.Volume{
			{
				Name: "tmp-file-storage",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		volumes = append(volumes, emptyDir...)
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if m.Spec.Worker.TopologySpreadConstraints != nil {
		topologySpreadConstraint = m.Spec.Worker.TopologySpreadConstraints
	}

	var dbHost, dbPort string
	if reflect.DeepEqual(m.Spec.Database.ExternalDB, repomanagerv1alpha1.ExternalDB{}) {
		containerPort := 0
		if m.Spec.Database.PostgresPort == 0 {
			containerPort = 5432
		} else {
			containerPort = m.Spec.Database.PostgresPort
		}
		dbHost = m.Name + "-database-svc"
		dbPort = strconv.Itoa(containerPort)
	} else {
		dbHost = m.Spec.Database.ExternalDB.PostgresHost
		dbPort = strconv.Itoa(m.Spec.Database.ExternalDB.PostgresPort)
	}

	envVars := []corev1.EnvVar{
		{Name: "POSTGRES_SERVICE_HOST", Value: dbHost},
		{Name: "POSTGRES_SERVICE_PORT", Value: dbPort},
	}

	if m.Spec.CacheEnabled {
		redisEnvVars := []corev1.EnvVar{
			{Name: "REDIS_SERVICE_HOST", Value: m.Name + "-redis-svc"},
			{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(m.Spec.RedisPort)},
		}
		envVars = append(envVars, redisEnvVars...)
	}

	if m.Spec.SigningSecret != "" {
		// for now, we are just dumping the error, but we should handle it
		signingKeyFingerprint, _ := r.getSigningKeyFingerprint(m.Spec.SigningSecret, m.Namespace)
		signingKeyEnvVars := []corev1.EnvVar{
			{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: signingKeyFingerprint},
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      m.Name + "-ansible-tmp",
			MountPath: "/.ansible/tmp",
		},
		{
			Name:      m.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      m.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
	}

	if m.Spec.SigningSecret != "" {
		signingSecretMount := []corev1.VolumeMount{
			{
				Name:      m.Name + "-signing-scripts",
				MountPath: "/var/lib/pulp/scripts",
				SubPath:   "scripts",
				ReadOnly:  true,
			},
			{
				Name:      m.Name + "-signing-galaxy",
				MountPath: "/etc/pulp/keys/signing_service.gpg",
				SubPath:   "signing_service.gpg",
				ReadOnly:  true,
			},
			{
				Name:      m.Name + "-signing-galaxy",
				MountPath: "/etc/pulp/keys/singing_service.asc",
				SubPath:   "signing_service.asc",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, signingSecretMount...)
	}

	if m.Spec.IsFileStorage {
		fileStorageMount := corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		}
		volumeMounts = append(volumeMounts, fileStorageMount)
	} else {
		emptyDir := []corev1.VolumeMount{
			{
				Name:      "tmp-file-storage",
				MountPath: "/var/lib/pulp/tmp",
			},
		}
		volumeMounts = append(volumeMounts, emptyDir...)
	}

	resources := m.Spec.Worker.ResourceRequirements

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-worker",
			Namespace: m.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Affinity:                  affinity,
					SecurityContext:           podSecurityContext,
					NodeSelector:              nodeSelector,
					Tolerations:               toleration,
					Volumes:                   volumes,
					ServiceAccountName:        "pulp-operator-go-controller-manager",
					TopologySpreadConstraints: topologySpreadConstraint,
					Containers: []corev1.Container{{
						Name:            "worker",
						Image:           m.Spec.Image + ":" + m.Spec.ImageVersion,
						ImagePullPolicy: corev1.PullPolicy(m.Spec.ImagePullPolicy),
						Args:            []string{"pulp-worker"},
						Env:             envVars,
						// LivenessProbe:  livenessProbe,
						// ReadinessProbe: readinessProbe,
						VolumeMounts: volumeMounts,
						Resources:    resources,
					}},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForPulpWorker returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpWorker(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-worker",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-worker-" + m.Name,
		"app.kubernetes.io/component":  "worker",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"app":                          "pulp-worker",
		"pulp_cr":                      m.Name,
	}
}
