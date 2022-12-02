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
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApiResource has the definition and function to provision api objects
type ApiResource struct {
	Definition ResourceDefinition
	Function   func(FunctionResources) client.Object
}

// pulpApiController provision and reconciles api objects
func (r *RepoManagerReconciler) pulpApiController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-API-Ready"
	funcResources := FunctionResources{ctx, pulp, log, r}

	// pulp-file-storage
	// the PVC will be created only if a StorageClassName is provided
	if storageClassProvided(pulp) {
		requeue, err := r.createPulpResource(ResourceDefinition{ctx, &corev1.PersistentVolumeClaim{}, pulp.Name + "-file-storage", "FileStorage", conditionType, pulp}, fileStoragePVC)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}

		// Reconcile PVC
		pvcFound := &corev1.PersistentVolumeClaim{}
		r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-file-storage", Namespace: pulp.Namespace}, pvcFound)
		expected_pvc := fileStoragePVC(funcResources)
		if !equality.Semantic.DeepDerivative(expected_pvc.(*corev1.PersistentVolumeClaim).Spec, pvcFound.Spec) {
			log.Info("The PVC has been modified! Reconciling ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "UpdatingFileStoragePVC", "Reconciling "+pulp.Name+"-file-storage PVC resource")
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling file storage PVC")
			err = r.Update(ctx, expected_pvc.(*corev1.PersistentVolumeClaim))
			if err != nil {
				log.Error(err, "Error trying to update the PVC object ... ")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingFileStoragePVC", "Failed to reconcile "+pulp.Name+"-file-storage PVC resource")
				r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile file storage PVC")
				return ctrl.Result{}, err
			}
			r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "File storage PVC reconciled")
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
		}
	}

	// if .spec.admin_password_secret is not defined, operator will default to pulp-admin-password
	adminSecretName := pulp.Name + "-admin-password"
	if len(pulp.Spec.AdminPasswordSecret) > 1 {
		adminSecretName = pulp.Spec.AdminPasswordSecret
	}

	// update pulp CR with container_token_secret secret value
	if len(pulp.Spec.ContainerTokenSecret) == 0 {
		patch := client.MergeFrom(pulp.DeepCopy())
		pulp.Spec.ContainerTokenSecret = pulp.Name + "-container-auth"
		r.Patch(ctx, pulp, patch)
	}

	// list of pulp-api resources that should be provisioned
	resources := []ApiResource{
		// pulp-server secret
		{Definition: ResourceDefinition{Context: ctx, Type: &corev1.Secret{}, Name: pulp.Name + "-server", Alias: "Server", ConditionType: conditionType, Pulp: pulp}, Function: pulpServerSecret},
		// pulp-db-fields-encryption secret
		{ResourceDefinition{ctx, &corev1.Secret{}, pulp.Name + "-db-fields-encryption", "DBFieldsEncryption", conditionType, pulp}, pulpDBFieldsEncryptionSecret},
		// pulp-admin-password secret
		{ResourceDefinition{ctx, &corev1.Secret{}, adminSecretName, "AdminPassword", conditionType, pulp}, pulpAdminPasswordSecret},
		// pulp-container-auth secret
		{ResourceDefinition{ctx, &corev1.Secret{}, pulp.Spec.ContainerTokenSecret, "ContainerAuth", conditionType, pulp}, pulpContainerAuth},
		// pulp-api deployment
		{ResourceDefinition{ctx, &appsv1.Deployment{}, pulp.Name + "-api", "Api", conditionType, pulp}, deploymentForPulpApi},
		// pulp-api-svc service
		{ResourceDefinition{ctx, &corev1.Service{}, pulp.Name + "-api-svc", "Api", conditionType, pulp}, serviceForAPI},
	}

	// create pulp-api resources
	for _, resource := range resources {
		requeue, err := r.createPulpResource(resource.Definition, resource.Function)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// update pulp CR admin-password secret with default name
	if err := r.updateCRField(ctx, pulp, "AdminPasswordSecret", pulp.Name+"-admin-password"); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure the deployment spec is as expected
	found := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, found)
	expected := deploymentForPulpApi(funcResources)
	if requeue, err := reconcileObject(funcResources, expected, found, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	// update pulp CR with default values
	if len(pulp.Spec.DBFieldsEncryptionSecret) == 0 {
		patch := client.MergeFrom(pulp.DeepCopy())
		pulp.Spec.DBFieldsEncryptionSecret = pulp.Name + "-db-fields-encryption"
		r.Patch(ctx, pulp, patch)
	}

	// Ensure the service spec is as expected
	apiSvc := &corev1.Service{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api-svc", Namespace: pulp.Namespace}, apiSvc)
	expectedSvc := serviceForAPI(funcResources)
	if requeue, err := reconcileObject(funcResources, expectedSvc, apiSvc, conditionType); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	return ctrl.Result{}, nil
}

// fileStoragePVC returns a PVC object
func fileStoragePVC(resources FunctionResources) client.Object {

	// Define the new PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-file-storage",
			Namespace: resources.Pulp.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       resources.Pulp.Spec.DeploymentType + "-storage",
				"app.kubernetes.io/instance":   resources.Pulp.Spec.DeploymentType + "-storage-" + resources.Pulp.Name,
				"app.kubernetes.io/component":  "storage",
				"app.kubernetes.io/part-of":    resources.Pulp.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": resources.Pulp.Spec.DeploymentType + "-operator",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(resources.Pulp.Spec.FileStorageSize),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(resources.Pulp.Spec.FileStorageAccessMode),
			},
			StorageClassName: &resources.Pulp.Spec.FileStorageClass,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, pvc, resources.RepoManagerReconciler.Scheme)
	return pvc
}

// deploymentForPulpApi returns a pulp-api Deployment object
func deploymentForPulpApi(resources FunctionResources) client.Object {
	replicas := resources.Pulp.Spec.Api.Replicas
	ls := labelsForPulpApi(resources.Pulp)

	affinity := &corev1.Affinity{}
	if resources.Pulp.Spec.Api.Affinity != nil {
		affinity = resources.Pulp.Spec.Api.Affinity
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := resources.Pulp.Spec.Api.Strategy
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
	if resources.Pulp.Spec.Api.NodeSelector != nil {
		nodeSelector = resources.Pulp.Spec.Api.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if resources.Pulp.Spec.Api.Tolerations != nil {
		toleration = resources.Pulp.Spec.Api.Tolerations
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if resources.Pulp.Spec.Api.TopologySpreadConstraints != nil {
		topologySpreadConstraint = resources.Pulp.Spec.Api.TopologySpreadConstraints
	}

	envVars := []corev1.EnvVar{
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(resources.Pulp.Spec.Api.GunicornTimeout)},
		{Name: "PULP_API_WORKERS", Value: strconv.Itoa(resources.Pulp.Spec.Api.GunicornWorkers)},
	}

	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(resources.Pulp.Spec.Database.ExternalDBSecret) == 0 {
		containerPort := 0
		if resources.Pulp.Spec.Database.PostgresPort == 0 {
			containerPort = 5432
		} else {
			containerPort = resources.Pulp.Spec.Database.PostgresPort
		}
		dbHost = resources.Pulp.Name + "-database-svc"
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
							Name: resources.Pulp.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	}

	// add cache configuration if enabled
	if resources.Pulp.Spec.Cache.Enabled {

		// if there is no ExternalCacheSecret defined, we should
		// use the redis instance provided by the operator
		if len(resources.Pulp.Spec.Cache.ExternalCacheSecret) == 0 {
			var cacheHost, cachePort string

			if resources.Pulp.Spec.Cache.RedisPort == 0 {
				cachePort = strconv.Itoa(6379)
			} else {
				cachePort = strconv.Itoa(resources.Pulp.Spec.Cache.RedisPort)
			}
			cacheHost = resources.Pulp.Name + "-redis-svc." + resources.Pulp.Namespace

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
								Name: resources.Pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_HOST",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PORT",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.Pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PORT",
						},
					},
				}, {
					Name: "REDIS_SERVICE_DB",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.Pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_DB",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.Pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PASSWORD",
						},
					},
				},
			}
			envVars = append(envVars, redisEnvVars...)
		}
	}

	if resources.Pulp.Spec.SigningSecret != "" {

		// for now, we are just dumping the error, but we should handle it
		signingKeyFingerprint, _ := resources.RepoManagerReconciler.getSigningKeyFingerprint(resources.Pulp.Spec.SigningSecret, resources.Pulp.Namespace)

		signingKeyEnvVars := []corev1.EnvVar{
			{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: signingKeyFingerprint},
			{Name: "COLLECTION_SIGNING_SERVICE", Value: getPulpSetting(resources.Pulp, "galaxy_collection_signing_service")},
			{Name: "CONTAINER_SIGNING_SERVICE", Value: getPulpSetting(resources.Pulp, "galaxy_container_signing_service")},
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}

	dbFieldsEncryptionSecret := ""
	if resources.Pulp.Spec.DBFieldsEncryptionSecret == "" {
		dbFieldsEncryptionSecret = resources.Pulp.Name + "-db-fields-encryption"
	} else {
		dbFieldsEncryptionSecret = resources.Pulp.Spec.DBFieldsEncryptionSecret
	}

	adminSecretName := resources.Pulp.Name + "-admin-password"
	if len(resources.Pulp.Spec.AdminPasswordSecret) > 1 {
		adminSecretName = resources.Pulp.Spec.AdminPasswordSecret
	}

	volumes := []corev1.Volume{
		{
			Name: resources.Pulp.Name + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.Pulp.Name + "-server",
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: adminSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: adminSecretName,
					Items: []corev1.KeyToPath{{
						Path: "admin-password",
						Key:  "password",
					}},
				},
			},
		},
		{
			Name: resources.Pulp.Name + "-db-fields-encryption",
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
	}

	_, storageType := controllers.MultiStorageConfigured(resources.Pulp, "Pulp")

	// if SC defined, we should use the PVC provisioned by the operator
	if storageType[0] == controllers.SCNameType {
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: resources.Pulp.Name + "-file-storage",
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
					ClaimName: resources.Pulp.Spec.PVC,
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
			{
				Name: "assets-file-storage",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		volumes = append(volumes, emptyDir...)
	}

	if resources.Pulp.Spec.SigningSecret != "" {
		signingSecretVolume := []corev1.Volume{
			{
				Name: resources.Pulp.Name + "-signing-scripts",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.SigningScriptsConfigmap,
						},
					},
				},
			},
			{
				Name: resources.Pulp.Name + "-signing-galaxy",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.Pulp.Spec.SigningSecret,
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

	var containerAuthSecretName string
	if resources.Pulp.Spec.ContainerTokenSecret != "" {
		containerAuthSecretName = resources.Pulp.Spec.ContainerTokenSecret
	} else {
		containerAuthSecretName = resources.Pulp.Name + "-container-auth"
	}

	containerTokenSecretVolume := corev1.Volume{
		Name: resources.Pulp.Name + "-container-auth-certs",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: containerAuthSecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  "container_auth_public_key.pem",
						Path: resources.Pulp.Spec.ContainerAuthPublicKey,
					},
					{
						Key:  "container_auth_private_key.pem",
						Path: resources.Pulp.Spec.ContainerAuthPrivateKey,
					},
				},
			},
		},
	}
	volumes = append(volumes, containerTokenSecretVolume)

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      resources.Pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      adminSecretName,
			MountPath: "/etc/pulp/pulp-admin-password",
			SubPath:   "admin-password",
			ReadOnly:  true,
		},
		{
			Name:      resources.Pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
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
			{
				Name:      "assets-file-storage",
				MountPath: "/var/lib/pulp/assets",
			},
		}
		volumeMounts = append(volumeMounts, emptyDir...)
	}

	if resources.Pulp.Spec.SigningSecret != "" {
		signingSecretMount := []corev1.VolumeMount{
			{
				Name:      resources.Pulp.Name + "-signing-scripts",
				MountPath: "/var/lib/pulp/scripts",
				SubPath:   "scripts",
				ReadOnly:  true,
			},
			{
				Name:      resources.Pulp.Name + "-signing-galaxy",
				MountPath: "/etc/pulp/keys/signing_service.gpg",
				SubPath:   "signing_service.gpg",
				ReadOnly:  true,
			},
			{
				Name:      resources.Pulp.Name + "-signing-galaxy",
				MountPath: "/etc/pulp/keys/singing_service.asc",
				SubPath:   "signing_service.asc",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, signingSecretMount...)
	}

	if resources.Pulp.Spec.ContainerTokenSecret != "" {
		containerTokenSecretMount := []corev1.VolumeMount{
			{
				Name:      resources.Pulp.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_private_key.pem",
				SubPath:   "container_auth_private_key.pem",
				ReadOnly:  true,
			},
			{
				Name:      resources.Pulp.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_public_key.pem",
				SubPath:   "container_auth_pulblic_key.pem",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, containerTokenSecretMount...)
	}

	// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
	if IsOpenShift {
		volumes, volumeMounts = mountCASpec(resources.Pulp, volumes, volumeMounts)
	}

	resourceRequirements := resources.Pulp.Spec.Api.ResourceRequirements

	readinessProbe := resources.Pulp.Spec.Api.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/usr/bin/readyz.py",
						getPulpSetting(resources.Pulp, "api_root") + "api/v3/status/",
					},
				},
			},
			FailureThreshold:    10,
			InitialDelaySeconds: 60,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		}
	}

	livenessProbe := resources.Pulp.Spec.Api.LivenessProbe
	if livenessProbe == nil {
		livenessProbe = &corev1.Probe{
			FailureThreshold: 5,
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: getPulpSetting(resources.Pulp, "api_root") + "api/v3/status/",
					Port: intstr.IntOrString{
						IntVal: 24817,
					},
					Scheme: corev1.URIScheme("HTTP"),
				},
			},
			InitialDelaySeconds: 120,
			PeriodSeconds:       20,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		}
	}

	// the following variables are defined to avoid issues with reconciliation
	restartPolicy := corev1.RestartPolicy("Always")
	terminationGracePeriodSeconds := int64(30)
	dnsPolicy := corev1.DNSPolicy("ClusterFirst")
	schedulerName := corev1.DefaultSchedulerName
	Image := os.Getenv("RELATED_IMAGE_PULP")
	if len(resources.Pulp.Spec.Image) > 0 && len(resources.Pulp.Spec.ImageVersion) > 0 {
		Image = resources.Pulp.Spec.Image + ":" + resources.Pulp.Spec.ImageVersion
	} else if Image == "" {
		Image = "quay.io/pulp/pulp-minimal:stable"
	}

	// deployment definition
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-api",
			Namespace: resources.Pulp.Namespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/no-node-affinity": "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       resources.Pulp.Spec.DeploymentType + "-api",
				"app.kubernetes.io/instance":   resources.Pulp.Spec.DeploymentType + "-api-" + resources.Pulp.Name,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    resources.Pulp.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": resources.Pulp.Spec.DeploymentType + "-operator",
				"app":                          "pulp-api",
				"pulp_cr":                      resources.Pulp.Name,
				"owner":                        "pulp-dev",
			},
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
					ServiceAccountName:        resources.Pulp.Name,
					TopologySpreadConstraints: topologySpreadConstraint,
					Containers: []corev1.Container{{
						Name:            "api",
						Image:           Image,
						ImagePullPolicy: corev1.PullPolicy(resources.Pulp.Spec.ImagePullPolicy),
						Args:            []string{"pulp-api"},
						Env:             envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 24817,
							Protocol:      "TCP",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						Resources:      resourceRequirements,
						VolumeMounts:   volumeMounts,
					}},

					/* the following configs are not defined on pulp-operator (ansible version)  */
					/* but i'll keep it here just in case we can manage to make deepequal usable */
					RestartPolicy:                 restartPolicy,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					DNSPolicy:                     dnsPolicy,
					SchedulerName:                 schedulerName,
				},
			},
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, dep, resources.RepoManagerReconciler.Scheme)
	return dep
}

// labelsForPulpApi returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpApi(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-api",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-api-" + m.Name,
		"app.kubernetes.io/component":  "api",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"app":                          "pulp-api",
		"pulp_cr":                      m.Name,
	}
}

// pulpServerSecret creates the pulp-server secret object which is used to
// populate the /etc/pulp/settings.py config file
func pulpServerSecret(resources FunctionResources) client.Object {

	var dbHost, dbPort, dbUser, dbPass, dbName, dbSSLMode string
	_, storageType := controllers.MultiStorageConfigured(resources.Pulp, "Pulp")

	// if there is no external database configuration get the databaseconfig from pulp-postgres-configuration secret
	if len(resources.Pulp.Spec.Database.ExternalDBSecret) == 0 {
		resources.Logger.Info("Retrieving Postgres credentials from "+resources.Pulp.Name+"-postgres-configuration secret", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		pgCredentials, err := resources.RepoManagerReconciler.retrieveSecretData(resources.Context, resources.Pulp.Name+"-postgres-configuration", resources.Pulp.Namespace, true, "username", "password", "database", "port", "sslmode")
		if err != nil {
			resources.Logger.Error(err, "Secret Not Found!", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		}
		dbHost = resources.Pulp.Name + "-database-svc"
		dbPort = pgCredentials["port"]
		dbUser = pgCredentials["username"]
		dbPass = pgCredentials["password"]
		dbName = pgCredentials["database"]
		dbSSLMode = pgCredentials["sslmode"]
	} else {
		resources.Logger.Info("Retrieving Postgres credentials from "+resources.Pulp.Spec.Database.ExternalDBSecret+" secret", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		externalPostgresData := []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USERNAME", "POSTGRES_PASSWORD", "POSTGRES_DB_NAME", "POSTGRES_SSLMODE"}
		pgCredentials, err := resources.RepoManagerReconciler.retrieveSecretData(resources.Context, resources.Pulp.Spec.Database.ExternalDBSecret, resources.Pulp.Namespace, true, externalPostgresData...)
		if err != nil {
			resources.Logger.Error(err, "Secret Not Found!", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		}
		dbHost = pgCredentials["POSTGRES_HOST"]
		dbPort = pgCredentials["POSTGRES_PORT"]
		dbUser = pgCredentials["POSTGRES_USERNAME"]
		dbPass = pgCredentials["POSTGRES_PASSWORD"]
		dbName = pgCredentials["POSTGRES_DB_NAME"]
		dbSSLMode = pgCredentials["POSTGRES_SSLMODE"]
	}

	// Handling user facing URLs
	rootUrl := getRootURL(resources)

	// default settings.py configuration
	var pulp_settings = `DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
GALAXY_CONTAINER_SIGNING_SERVICE = "container-default"
ANSIBLE_API_HOSTNAME = "` + rootUrl + `"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys/"
CONTENT_ORIGIN = "` + rootUrl + `"
DATABASES = {
	'default': {
		'HOST': '` + dbHost + `',
		'ENGINE': 'django.db.backends.postgresql_psycopg2',
		'NAME': '` + dbName + `',
		'USER': '` + dbUser + `',
		'PASSWORD': '` + dbPass + `',
		'PORT': '` + dbPort + `',
		'CONN_MAX_AGE': 0,
		'OPTIONS': { 'sslmode': '` + dbSSLMode + `' },
	}
}
GALAXY_FEATURE_FLAGS = {
	'execution_environments': 'True',
}
PRIVATE_KEY_PATH = "/etc/pulp/keys/container_auth_private_key.pem"
PUBLIC_KEY_PATH = "/etc/pulp/keys/container_auth_public_key.pem"
STATIC_ROOT = "/var/lib/operator/static/"
TOKEN_AUTH_DISABLED = "False"
TOKEN_SERVER = "http://` + resources.Pulp.Name + `-api-svc.` + resources.Pulp.Namespace + `.svc.cluster.local:24817/token/"
TOKEN_SIGNATURE_ALGORITHM = "ES256"
`

	pulp_settings = pulp_settings + fmt.Sprintln("API_ROOT = \"/pulp/\"")

	// add cache settings
	if resources.Pulp.Spec.Cache.Enabled {

		var cacheHost, cachePort, cachePassword, cacheDB string

		// if there is no ExternalCacheSecret defined, we should
		// use the redis instance provided by the operator
		if len(resources.Pulp.Spec.Cache.ExternalCacheSecret) == 0 {
			if resources.Pulp.Spec.Cache.RedisPort == 0 {
				cachePort = strconv.Itoa(6379)
			} else {
				cachePort = strconv.Itoa(resources.Pulp.Spec.Cache.RedisPort)
			}
			cacheHost = resources.Pulp.Name + "-redis-svc." + resources.Pulp.Namespace
		} else {
			// retrieve the connection data from ExternalCacheSecret secret
			externalCacheData := []string{"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB"}
			externalCacheConfig, _ := resources.RepoManagerReconciler.retrieveSecretData(context.TODO(), resources.Pulp.Spec.Cache.ExternalCacheSecret, resources.Pulp.Namespace, true, externalCacheData...)
			cacheHost = externalCacheConfig["REDIS_HOST"]
			cachePort = externalCacheConfig["REDIS_PORT"]
			cachePassword = externalCacheConfig["REDIS_PASSWORD"]
			cacheDB = externalCacheConfig["REDIS_DB"]
		}

		cacheSettings := `CACHE_ENABLED = "True"
REDIS_HOST =  "` + cacheHost + `"
REDIS_PORT =  "` + cachePort + `"
REDIS_PASSWORD = "` + cachePassword + `"
REDIS_DB = "` + cacheDB + `"
`
		pulp_settings = pulp_settings + cacheSettings
	}

	// if an Azure Blob is defined in Pulp CR we should add the
	// credentials from azure secret into settings.py
	if storageType[0] == controllers.AzureObjType {
		resources.Logger.Info("Retrieving Azure data from " + resources.Pulp.Spec.ObjectStorageAzureSecret)
		storageData, err := resources.RepoManagerReconciler.retrieveSecretData(resources.Context, resources.Pulp.Spec.ObjectStorageAzureSecret, resources.Pulp.Namespace, true, "azure-account-name", "azure-account-key", "azure-container", "azure-container-path", "azure-connection-string")
		if err != nil {
			resources.Logger.Error(err, "Secret Not Found!", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Spec.ObjectStorageAzureSecret)
		}
		pulp_settings = pulp_settings + `AZURE_CONNECTION_STRING = '` + storageData["azure-connection-string"] + `'
AZURE_LOCATION = '` + storageData["azure-container-path"] + `'
AZURE_ACCOUNT_NAME = '` + storageData["azure-account-name"] + `'
AZURE_ACCOUNT_KEY = '` + storageData["azure-account-key"] + `'
AZURE_CONTAINER = '` + storageData["azure-container"] + `'
AZURE_URL_EXPIRATION_SECS = 60
AZURE_OVERWRITE_FILES = "True"
DEFAULT_FILE_STORAGE = "storages.backends.azure_storage.AzureStorage"
`
	}

	// if a S3 is defined in Pulp CR we should add the
	// credentials from aws secret into settings.py
	if storageType[0] == controllers.S3ObjType {
		resources.Logger.Info("Retrieving S3 data from " + resources.Pulp.Spec.ObjectStorageS3Secret)
		storageData, err := resources.RepoManagerReconciler.retrieveSecretData(resources.Context, resources.Pulp.Spec.ObjectStorageS3Secret, resources.Pulp.Namespace, true, "s3-access-key-id", "s3-secret-access-key", "s3-bucket-name", "s3-region")
		if err != nil {
			resources.Logger.Error(err, "Secret Not Found!", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Spec.ObjectStorageS3Secret)
		}

		optionalKey, _ := resources.RepoManagerReconciler.retrieveSecretData(resources.Context, resources.Pulp.Spec.ObjectStorageS3Secret, resources.Pulp.Namespace, false, "s3-endpoint")
		if len(optionalKey["s3-endpoint"]) > 0 {
			pulp_settings = pulp_settings + fmt.Sprintf("AWS_S3_ENDPOINT_URL = \"%v\"\n", optionalKey["s3-endpoint"])
		}

		pulp_settings = pulp_settings + `AWS_ACCESS_KEY_ID = '` + storageData["s3-access-key-id"] + `'
AWS_SECRET_ACCESS_KEY = '` + storageData["s3-secret-access-key"] + `'
AWS_STORAGE_BUCKET_NAME = '` + storageData["s3-bucket-name"] + `'
AWS_S3_REGION_NAME = '` + storageData["s3-region"] + `'
AWS_DEFAULT_ACL = "@none None"
S3_USE_SIGV4 = "True"
AWS_S3_SIGNATURE_VERSION = "s3v4"
AWS_S3_ADDRESSING_STYLE = "path"
DEFAULT_FILE_STORAGE = "storages.backends.s3boto3.S3Boto3Storage"
MEDIA_ROOT = ""
`
	}

	// configure settings.py with keycloak integration variables
	if len(resources.Pulp.Spec.SSOSecret) > 0 {
		resources.RepoManagerReconciler.ssoConfig(resources.Context, resources.Pulp, &pulp_settings)
	}

	pulp_settings = addCustomPulpSettings(resources.Pulp, pulp_settings)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-server",
			Namespace: resources.Pulp.Namespace,
		},
		StringData: map[string]string{
			"settings.py": pulp_settings,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, sec, resources.RepoManagerReconciler.Scheme)
	return sec
}

// pulp-db-fields-encryption secret
func pulpDBFieldsEncryptionSecret(resources FunctionResources) client.Object {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-db-fields-encryption",
			Namespace: resources.Pulp.Namespace,
		},
		StringData: map[string]string{
			"database_fields.symmetric.key": createFernetKey(),
		},
	}
	return sec
}

// pulp-admin-passowrd
func pulpAdminPasswordSecret(resources FunctionResources) client.Object {

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-admin-password",
			Namespace: resources.Pulp.Namespace,
		},
		StringData: map[string]string{
			"password": createPwd(32),
		},
	}
	ctrl.SetControllerReference(resources.Pulp, sec, resources.RepoManagerReconciler.Scheme)

	return sec
}

// pulp-container-auth
func pulpContainerAuth(resources FunctionResources) client.Object {

	privKey, pubKey := genTokenAuthKey()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-container-auth",
			Namespace: resources.Pulp.Namespace,
		},
		StringData: map[string]string{
			"container_auth_private_key.pem": privKey,
			"container_auth_public_key.pem":  pubKey,
		},
	}
}

// serviceForAPI returns a service object for pulp-api
func serviceForAPI(resources FunctionResources) client.Object {

	svc := serviceAPIObject(resources.Pulp.Name, resources.Pulp.Namespace, resources.Pulp.Spec.DeploymentType)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, svc, resources.RepoManagerReconciler.Scheme)
	return svc
}

func serviceAPIObject(name, namespace, deployment_type string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-api-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       deployment_type + "-api",
				"app.kubernetes.io/instance":   deployment_type + "-api-" + name,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    deployment_type,
				"app.kubernetes.io/managed-by": deployment_type + "-operator",
				"app":                          "pulp-api",
				"pulp_cr":                      name,
			},
		},
		Spec: serviceAPISpec(name, namespace, deployment_type),
	}
}

// api service spec
func serviceAPISpec(name, namespace, deployment_type string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24817}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Name:       "api-24817",
			Port:       24817,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app.kubernetes.io/name":       deployment_type + "-api",
			"app.kubernetes.io/instance":   deployment_type + "-api-" + name,
			"app.kubernetes.io/component":  "api",
			"app.kubernetes.io/part-of":    deployment_type,
			"app.kubernetes.io/managed-by": deployment_type + "-operator",
			"app":                          "pulp-api",
			"pulp_cr":                      name,
		},
		SessionAffinity:          serviceAffinity,
		Type:                     serviceType,
		PublishNotReadyAddresses: true,
	}
}

// storageClassProvided returns true if a StorageClass is provided in Pulp CR
func storageClassProvided(pulp *repomanagerv1alpha1.Pulp) bool {
	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	return storageType[0] == controllers.SCNameType
}
