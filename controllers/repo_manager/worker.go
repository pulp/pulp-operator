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

package repo_manager

import (
	"context"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

func (r *RepoManagerReconciler) pulpWorkerController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Worker-Ready"

	// Worker Deployment
	workerDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-worker", Namespace: pulp.Namespace}, workerDeployment)
	newWorkerDeployment := r.deploymentForPulpWorker(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "CreatingWorkerDeployment", "Creating "+pulp.Name+"-worker deployment resource")
		err = r.Create(ctx, newWorkerDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingWorkerDeployment", "Failed to create "+pulp.Name+"-worker deployment resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create new Worker Deployment")
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Worker Deployment created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Worker Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if deploymentModified(newWorkerDeployment, workerDeployment) {
		log.Info("The Worker Deployment has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "UpdatingWorkerDeployment", "Reconciling "+pulp.Name+"-worker deployment resource")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling Worker Deployment")
		err = r.Update(ctx, newWorkerDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the Worker Deployment object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingWorkerDeployment", "Failed to reconcile "+pulp.Name+"-worker deployment resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile Worker Deployment")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Worker Deployment reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	// we should only update the status when Worker-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		r.updateStatus(ctx, pulp, metav1.ConditionTrue, conditionType, "WorkerTasksFinished", "All Worker tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "WorkerReady", "All Worker tasks ran successfully")
	}
	return ctrl.Result{}, nil
}

// deploymentForPulpWorker returns a pulp-worker Deployment object
func (r *RepoManagerReconciler) deploymentForPulpWorker(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {
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

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := m.Spec.Worker.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	// pulp image is built to run with user 0
	// we are enforcing the containers to run as 1000
	runAsUser := int64(700)
	fsGroup := int64(700)
	podSecurityContext := &corev1.PodSecurityContext{}
	IsOpenShift, _ := controllers.IsOpenShift()
	if !IsOpenShift {
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

	_, storageType := controllers.MultiStorageConfigured(m, "Pulp")

	// if SC defined, we should use the PVC provisioned by the operator
	if storageType[0] == controllers.SCNameType {
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.Name + "-file-storage",
				},
			},
		}
		volumes = append(volumes, fileStorage)

		// if .spec.Api.PVC defined we should use the PVC provisioned by user
	} else if storageType[0] == controllers.PVCType {
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.Spec.PVC,
				},
			},
		}
		volumes = append(volumes, fileStorage)

		// if there is no SC nor PVC nor object storage defined we will mount an emptyDir
	} else if storageType[0] == controllers.EmptyDirType {
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

	envVars := []corev1.EnvVar{}
	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(m.Spec.Database.ExternalDBSecret) == 0 {
		containerPort := 0
		if m.Spec.Database.PostgresPort == 0 {
			containerPort = 5432
		} else {
			containerPort = m.Spec.Database.PostgresPort
		}
		dbHost = m.Name + "-database-svc"
		dbPort = strconv.Itoa(containerPort)

		postgresEnvVars := []corev1.EnvVar{
			{Name: "POSTGRES_SERVICE_HOST", Value: dbHost},
			{Name: "POSTGRES_SERVICE_PORT", Value: dbPort},
		}
		envVars = append(envVars, postgresEnvVars...)
	} else {
		postgresEnvVars := []corev1.EnvVar{
			{
				Name: "POSTGRES_SERVICE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	}

	// add cache configuration if enabled
	if m.Spec.Cache.Enabled {

		// if there is no ExternalCacheSecret defined, we should
		// use the redis instance provided by the operator
		if len(m.Spec.Cache.ExternalCacheSecret) == 0 {
			var cacheHost, cachePort string

			if m.Spec.Cache.RedisPort == 0 {
				cachePort = strconv.Itoa(6379)
			} else {
				cachePort = strconv.Itoa(m.Spec.Cache.RedisPort)
			}
			cacheHost = m.Name + "-redis-svc." + m.Namespace

			redisEnvVars := []corev1.EnvVar{
				{Name: "REDIS_SERVICE_HOST", Value: cacheHost},
				{Name: "REDIS_SERVICE_PORT", Value: cachePort},
			}
			envVars = append(envVars, redisEnvVars...)
		} else {
			redisEnvVars := []corev1.EnvVar{
				{
					Name: "REDIS_SERVICE_HOST",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: m.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_HOST",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PORT",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: m.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PORT",
						},
					},
				}, {
					Name: "REDIS_SERVICE_DB",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: m.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_DB",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: m.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PASSWORD",
						},
					},
				},
			}
			envVars = append(envVars, redisEnvVars...)
		}
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

	// we will mount file-storage if a storageclass or a pvc was provided
	if storageType[0] == controllers.SCNameType || storageType[0] == controllers.PVCType {
		fileStorageMount := corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		}
		volumeMounts = append(volumeMounts, fileStorageMount)

		// if no file-storage nor object storage were provided we will mount the emptyDir
	} else if storageType[0] == controllers.EmptyDirType {
		emptyDir := []corev1.VolumeMount{
			{
				Name:      "tmp-file-storage",
				MountPath: "/var/lib/pulp/tmp",
			},
		}
		volumeMounts = append(volumeMounts, emptyDir...)
	}

	// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
	if IsOpenShift {
		volumes, volumeMounts = mountCASpec(m, volumes, volumeMounts)
	}

	resources := m.Spec.Worker.ResourceRequirements
	Image := os.Getenv("RELATED_IMAGE_PULP")
	if len(m.Spec.Image) > 0 && len(m.Spec.ImageVersion) > 0 {
		Image = m.Spec.Image + ":" + m.Spec.ImageVersion
	} else if Image == "" {
		Image = "quay.io/pulp/pulp:stable"
	}

	readinessProbe := m.Spec.Worker.ReadinessProbe
	livenessProbe := m.Spec.Worker.LivenessProbe

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-worker",
			Namespace: m.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: strategy,
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
					ServiceAccountName:        m.Name,
					TopologySpreadConstraints: topologySpreadConstraint,
					Containers: []corev1.Container{{
						Name:            "worker",
						Image:           Image,
						ImagePullPolicy: corev1.PullPolicy(m.Spec.ImagePullPolicy),
						Args:            []string{"pulp-worker"},
						Env:             envVars,
						LivenessProbe:   livenessProbe,
						ReadinessProbe:  readinessProbe,
						VolumeMounts:    volumeMounts,
						Resources:       resources,
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
