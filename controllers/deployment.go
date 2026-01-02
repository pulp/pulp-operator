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
	"os"
	"reflect"
	"strconv"
	"strings"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers/settings"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CommonDeployment has the common definition for all pulpcore deployments
type CommonDeployment struct {
	replicas                          *int32
	podLabels                         map[string]string
	podSelectorLabels                 map[string]string
	deploymentLabels                  map[string]string
	affinity                          *corev1.Affinity
	strategy                          appsv1.DeploymentStrategy
	podSecurityContext                *corev1.PodSecurityContext
	nodeSelector                      map[string]string
	toleration                        []corev1.Toleration
	topologySpreadConstraint          []corev1.TopologySpreadConstraint
	envVars                           []corev1.EnvVar
	volumes                           []corev1.Volume
	volumeMounts                      []corev1.VolumeMount
	resourceRequirements              corev1.ResourceRequirements
	readinessProbe                    *corev1.Probe
	livenessProbe                     *corev1.Probe
	image                             string
	containers                        []corev1.Container
	podAnnotations                    map[string]string
	deploymentAnnotations             map[string]string
	restartPolicy                     corev1.RestartPolicy
	terminationPeriod                 *int64
	dnsPolicy                         corev1.DNSPolicy
	schedulerName                     string
	initContainerEnvVars              []corev1.EnvVar
	initContainerResourceRequirements corev1.ResourceRequirements
	initContainerVolumeMounts         []corev1.VolumeMount
	initContainerImage                string
	initContainers                    []corev1.Container
}

// Deploy returns a common Deployment object that can be used by any pulpcore component
func (d CommonDeployment) Deploy(resources any, pulpcoreType settings.PulpcoreType) client.Object {
	pulp := resources.(FunctionResources).Pulp
	d.build(resources, pulpcoreType)

	// deployment definition
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pulpcoreType.DeploymentName(pulp.Name),
			Namespace:   pulp.Namespace,
			Annotations: d.deploymentAnnotations,
			Labels:      d.deploymentLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: d.replicas,
			Strategy: d.strategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: d.podSelectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.podLabels,
					Annotations: d.podAnnotations,
				},
				Spec: corev1.PodSpec{
					Affinity:                      d.affinity,
					SecurityContext:               d.podSecurityContext,
					NodeSelector:                  d.nodeSelector,
					Tolerations:                   d.toleration,
					Volumes:                       d.volumes,
					ServiceAccountName:            settings.PulpServiceAccount(pulp.Name),
					TopologySpreadConstraints:     d.topologySpreadConstraint,
					InitContainers:                d.initContainers,
					Containers:                    d.containers,
					RestartPolicy:                 d.restartPolicy,
					TerminationGracePeriodSeconds: d.terminationPeriod,
					DNSPolicy:                     d.dnsPolicy,
					SchedulerName:                 d.schedulerName,
				},
			},
		},
	}

	AddHashLabel(resources.(FunctionResources), dep)
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, dep, resources.(FunctionResources).Scheme)
	return dep
}

// DeploymentAPICommon is the common pulpcore-api Deployment definition
type DeploymentAPICommon struct {
	CommonDeployment
}

// Deploy returns a pulp-api Deployment object
func (d DeploymentAPICommon) Deploy(resources any) client.Object {
	return d.CommonDeployment.Deploy(resources, settings.API)
}

// DeploymentContentCommon is the common pulpcore-content Deployment definition
type DeploymentContentCommon struct {
	CommonDeployment
}

// Deploy returns a pulp-content Deployment object
func (d DeploymentContentCommon) Deploy(resources any) client.Object {
	return d.CommonDeployment.Deploy(resources, settings.CONTENT)
}

// DeploymentWorkerCommon is the common pulpcore-worker Deployment definition
type DeploymentWorkerCommon struct {
	CommonDeployment
}

// Deploy returns a pulp-worker Deployment object
func (d DeploymentWorkerCommon) Deploy(resources any) client.Object {
	return d.CommonDeployment.Deploy(resources, settings.WORKER)
}

// setReplicas defines the number of pod replicas
// When HPA is enabled, replicas should not be set in the deployment spec
// to allow HPA to manage the replica count
func (d *CommonDeployment) setReplicas(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	// Check if HPA is enabled for this component
	hpaConfig := getHPAConfigForDeployment(pulp, pulpcoreType)
	if hpaConfig != nil && hpaConfig.Enabled {
		d.replicas = nil
		return
	}

	// Use the static replica count from the spec
	val := int32(reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("Replicas").Int())
	d.replicas = &val
}

// getHPAConfigForDeployment retrieves HPA configuration for a specific component
func getHPAConfigForDeployment(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) *pulpv1.HPA {
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType))
	if !specField.IsValid() {
		return nil
	}

	hpaField := specField.FieldByName("HPA")
	if !hpaField.IsValid() || hpaField.IsNil() {
		return nil
	}

	return hpaField.Interface().(*pulpv1.HPA)
}

// setLabels defines the pod and deployment labels
func (d *CommonDeployment) setLabels(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	d.podLabels = settings.PulpcorePodLabels(pulp, pulpcoreType)
	d.podSelectorLabels = settings.PulpcoreLabels(pulp, pulpcoreType)
	d.deploymentLabels = settings.PulpcoreLabels(pulp, pulpcoreType)
}

// setAffinity defines the affinity rules
func (d *CommonDeployment) setAffinity(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	affinity := &corev1.Affinity{}
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("Affinity").Interface().(*corev1.Affinity)
	if specField != nil {
		affinity = specField
	}
	d.affinity = affinity
}

// setStrategy defines the deployment strategy to use to replace existing pods with new ones
func (d *CommonDeployment) setStrategy(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("Strategy").Interface().(appsv1.DeploymentStrategy)
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	d.strategy = strategy
}

// setPodSecurityContext defines the pod-level security attributes
func (d *CommonDeployment) setPodSecurityContext() {
	runAsUser := int64(700)
	fsGroup := int64(700)
	d.podSecurityContext = &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
		FSGroup:   &fsGroup,
	}
}

// setNodeSelector defines the selectors to schedule the pod on a node
func (d *CommonDeployment) setNodeSelector(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	nodeSelector := map[string]string{}
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("NodeSelector").Interface().(map[string]string)
	if specField != nil {
		nodeSelector = specField
	}
	d.nodeSelector = nodeSelector
}

// setTolerations defines the pod tolerations
func (d *CommonDeployment) setTolerations(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	toleration := []corev1.Toleration{}
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("Tolerations").Interface().([]corev1.Toleration)
	if specField != nil {
		toleration = specField
	}
	d.toleration = append([]corev1.Toleration(nil), toleration...)
}

// setTopologySpreadConstraints defines how to spread pods across topology
func (d *CommonDeployment) setTopologySpreadConstraints(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("TopologySpreadConstraints").Interface().([]corev1.TopologySpreadConstraint)
	if specField != nil {
		topologySpreadConstraint = specField
	}
	d.topologySpreadConstraint = append([]corev1.TopologySpreadConstraint(nil), topologySpreadConstraint...)
}

// setEnvVars defines the list of containers' environment variables
func (d *CommonDeployment) setEnvVars(resources any, pulpcoreType settings.PulpcoreType) {
	pulp := resources.(FunctionResources).Pulp
	pulpcoreTypeField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType))
	ctx := resources.(FunctionResources).Context

	envVars := SetPulpcoreCustomEnvVars(*pulp, pulpcoreType)

	if pulpcoreType != settings.WORKER {
		// get gunicornWorkers definition from CR
		gunicornWorkers := strconv.FormatInt(pulpcoreTypeField.FieldByName("GunicornWorkers").Int(), 10)
		if gunicornWorkers == "0" { // set default value if none provided
			gunicornWorkers = "2"
		}

		// get gunicornTimeout definition from CR
		gunicornTimeout := strconv.FormatInt(pulpcoreTypeField.FieldByName("GunicornTimeout").Int(), 10)
		if gunicornTimeout == "0" { // set default value if none provided
			gunicornTimeout = "90"
		}

		// get gunicornAccessLogformat defintion from CR
		gunicornAccessLogformat := pulpcoreTypeField.FieldByName("GunicornAccessLogformat").String()
		if gunicornAccessLogformat == "" {
			gunicornAccessLogformat = `pulp [%({correlation-id}o)s]: %(h)s %(l)s %(u)s %(t)s "%(r)s" %(s)s %(b)s "%(f)s" "%(a)s"`
		}

		gunicornEnvVars := []corev1.EnvVar{
			{Name: "PULP_GUNICORN_TIMEOUT", Value: gunicornTimeout},
			{Name: "PULP_" + strings.ToUpper(string(pulpcoreType)) + "_WORKERS", Value: gunicornWorkers},
			{Name: "PULP_GUNICORN_ACCESS_LOGFORMAT", Value: gunicornAccessLogformat},
		}
		envVars = append(envVars, gunicornEnvVars...)
	}

	// add postgres env vars
	envVars = append(envVars, GetPostgresEnvVars(*pulp)...)

	// add cache configuration if enabled
	if pulp.Spec.Cache.Enabled {

		// if there is no ExternalCacheSecret defined, we should
		// use the redis instance provided by the operator
		if len(pulp.Spec.Cache.ExternalCacheSecret) == 0 {
			var cacheHost, cachePort string

			if pulp.Spec.Cache.RedisPort == 0 {
				cachePort = strconv.Itoa(6379)
			} else {
				cachePort = strconv.Itoa(pulp.Spec.Cache.RedisPort)
			}
			cacheHost = pulp.Name + "-redis-svc." + pulp.Namespace

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
								Name: pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_HOST",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PORT",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PORT",
						},
					},
				}, {
					Name: "REDIS_SERVICE_DB",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_DB",
						},
					},
				}, {
					Name: "REDIS_SERVICE_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: pulp.Spec.Cache.ExternalCacheSecret,
							},
							Key: "REDIS_PASSWORD",
						},
					},
				},
			}
			envVars = append(envVars, redisEnvVars...)
		}
	}

	if pulp.Spec.SigningSecret != "" {

		// for now, we are just dumping the error, but we should handle it
		signingKeyFingerprint, _ := GetSigningKeyFingerprint(ctx, resources.(FunctionResources).Client, pulp.Spec.SigningSecret, pulp.Namespace)

		signingKeyEnvVars := []corev1.EnvVar{
			{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: signingKeyFingerprint},
			{Name: "HOME", Value: "/var/lib/pulp"},
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}
	d.envVars = append([]corev1.EnvVar(nil), envVars...)
}

// setInitContainerEnvVars defines the list of init-containers' environment variables
func (d *CommonDeployment) setInitContainerEnvVars(resources any, pulpcoreType settings.PulpcoreType) {
	pulp := resources.(FunctionResources).Pulp
	d.initContainerEnvVars = append(GetPostgresEnvVars(*pulp), SetPulpcoreCustomEnvVars(*pulp, pulpcoreType)...)
}

// GetPostgresEnvVars return the list of postgres environment variables to use in containers
func GetPostgresEnvVars(pulp pulpv1.Pulp) (envVars []corev1.EnvVar) {
	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(pulp.Spec.Database.ExternalDBSecret) == 0 {
		containerPort := 0
		if pulp.Spec.Database.PostgresPort == 0 {
			containerPort = 5432
		} else {
			containerPort = pulp.Spec.Database.PostgresPort
		}
		dbHost = pulp.Name + "-database-svc"
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
							Name: pulp.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: pulp.Spec.Database.ExternalDBSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	}
	return envVars
}

// GetAdminSecretName retrieves pulp admin user password
func GetAdminSecretName(pulp pulpv1.Pulp) string {
	return pulp.Spec.AdminPasswordSecret
}

// GetDBFieldsEncryptionSecret returns the name of DBFieldsEncryption Secret
func GetDBFieldsEncryptionSecret(pulp pulpv1.Pulp) string {
	return pulp.Spec.DBFieldsEncryptionSecret
}

// setVolumes defines the list of pod volumes
func (d *CommonDeployment) setVolumes(resources any, pulpcoreType settings.PulpcoreType) {
	pulp := *resources.(FunctionResources).Pulp
	dbFieldsEncryptionSecret := GetDBFieldsEncryptionSecret(pulp)
	volumes := []corev1.Volume{
		{
			Name: pulp.Name + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: settings.PulpServerSecret(pulp.Name),
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
					}},
				},
			},
		},
		{
			Name: pulp.Name + "-db-fields-encryption",
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

	// only worker pods need to mount ansible dir
	if pulpcoreType == settings.WORKER {
		ansibleVolume := corev1.Volume{
			Name: pulp.Name + "-ansible-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		volumes = append(volumes, ansibleVolume)

		// mount wait_on_postgres.py if ipv6 is disabled
		if Ipv6Disabled(pulp) {
			defaultMode := int32(0755)
			waitOnPostgres := corev1.Volume{
				Name: pulp.Name + "-worker-probe",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						DefaultMode: &defaultMode,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: pulp.Name + "-worker-probe",
						},
						Items: []corev1.KeyToPath{
							{Key: "wait_on_postgres.py", Path: "wait_on_postgres.py"},
						},
					},
				},
			}
			volumes = append(volumes, waitOnPostgres)
		}
	}

	// worker and content pods don't need to mount the admin secret
	if pulpcoreType == settings.API {
		adminSecretName := GetAdminSecretName(pulp)
		volume := corev1.Volume{
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
		}
		volumes = append(volumes, volume)
	}

	storageType := GetStorageType(pulp)
	if storageType[0] == SCNameType { // if SC defined, we should use the PVC provisioned by the operator
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: settings.DefaultPulpFileStorage(pulp.Name),
				},
			},
		}
		volumes = append(volumes, fileStorage)
	} else if storageType[0] == PVCType { // if .spec.Api.PVC defined we should use the PVC provisioned by user
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pulp.Spec.PVC,
				},
			},
		}
		volumes = append(volumes, fileStorage)
	}

	volumes = signingMetadataVolumes(resources, storageType, volumes)

	// only api pods need the container-auth-certs
	if pulpcoreType == settings.API {
		containerAuthSecretName := pulp.Spec.ContainerTokenSecret
		containerTokenSecretVolume := corev1.Volume{
			Name: pulp.Name + "-container-auth-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: containerAuthSecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  "container_auth_public_key.pem",
							Path: pulp.Spec.ContainerAuthPublicKey,
						},
						{
							Key:  "container_auth_private_key.pem",
							Path: pulp.Spec.ContainerAuthPrivateKey,
						},
					},
				},
			},
		}
		volumes = append(volumes, containerTokenSecretVolume)
	}
	d.volumes = append([]corev1.Volume(nil), volumes...)
}

// signingMetadataVolumes defines the volumes for the signing metadata services
func signingMetadataVolumes(resources any, storageType []string, volumes []corev1.Volume) []corev1.Volume {
	pulp := *resources.(FunctionResources).Pulp
	if pulp.Spec.SigningSecret != "" {
		if storageType[0] != SCNameType && storageType[0] != PVCType {
			ephemeralGpg := corev1.Volume{
				Name: "ephemeral-gpg",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
			volumes = append(volumes, ephemeralGpg)
		}

		ctx := resources.(FunctionResources).Context
		client := resources.(FunctionResources).Client
		secretName := pulp.Spec.SigningScripts
		secret := &corev1.Secret{}
		client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, secret)

		secretItems := []corev1.KeyToPath{}
		if DeployCollectionSign(*secret) {
			item := corev1.KeyToPath{Key: settings.CollectionSigningScriptName, Path: settings.CollectionSigningScriptName}
			secretItems = append(secretItems, item)
		}
		if DeployContainerSign(*secret) {
			item := corev1.KeyToPath{Key: settings.ContainerSigningScriptName, Path: settings.ContainerSigningScriptName}
			secretItems = append(secretItems, item)
		}
		if DeployAptSign(*secret) {
			item := corev1.KeyToPath{Key: settings.AptSigningScriptName, Path: settings.AptSigningScriptName}
			secretItems = append(secretItems, item)
		}
		if DeployRpmSign(*secret) {
			item := corev1.KeyToPath{Key: settings.RpmSigningScriptName, Path: settings.RpmSigningScriptName}
			secretItems = append(secretItems, item)
		}
		volumePermissions := int32(0755)
		signingSecretVolume := []corev1.Volume{
			{
				Name: "gpg-keys",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: pulp.Spec.SigningSecret,
						Items: []corev1.KeyToPath{
							{
								Key:  "signing_service.gpg",
								Path: "signing_service.gpg",
							},
						},
					},
				},
			},
			{
				Name: pulp.Name + "-signing-scripts",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  pulp.Spec.SigningScripts,
						Items:       secretItems,
						DefaultMode: &volumePermissions,
					},
				},
			},
		}
		volumes = append(volumes, signingSecretVolume...)
	}

	return volumes
}

// setVolumeMounts defines the list containers volumes mount points
func (d *CommonDeployment) setVolumeMounts(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
	}

	// only worker pods need to mount ansible dir
	if pulpcoreType == settings.WORKER {
		ansibleVolume := corev1.VolumeMount{Name: pulp.Name + "-ansible-tmp", MountPath: "/.ansible/tmp"}
		volumeMounts = append(volumeMounts, ansibleVolume)

		if Ipv6Disabled(pulp) {
			waitOnPostgres := corev1.VolumeMount{
				Name:      pulp.Name + "-worker-probe",
				MountPath: "/usr/bin/wait_on_postgres.py",
				SubPath:   "wait_on_postgres.py",
			}
			volumeMounts = append(volumeMounts, waitOnPostgres)
		}
	}

	// worker and content pods don't need to mount the admin secret
	if pulpcoreType == settings.API {
		adminSecretName := GetAdminSecretName(pulp)
		adminSecret := corev1.VolumeMount{
			Name:      adminSecretName,
			MountPath: "/etc/pulp/pulp-admin-password",
			SubPath:   "admin-password",
			ReadOnly:  true,
		}
		volumeMounts = append(volumeMounts, adminSecret)
	}

	storageType := GetStorageType(pulp)
	if storageType[0] == SCNameType || storageType[0] == PVCType { // we will mount file-storage if a storageclass or a pvc was provided
		fileStorageMount := corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		}
		volumeMounts = append(volumeMounts, fileStorageMount)
	}

	if pulp.Spec.SigningSecret != "" {
		if storageType[0] != SCNameType && storageType[0] != PVCType {
			signingSecretMount := corev1.VolumeMount{
				Name:      "ephemeral-gpg",
				MountPath: "/var/lib/pulp/.gnupg",
			}
			volumeMounts = append(volumeMounts, signingSecretMount)
		}

		for _, volume := range d.volumes {
			if volume.Name == pulp.Name+"-signing-scripts" {
				for _, script := range volume.VolumeSource.Secret.Items {
					signingSecretMount := corev1.VolumeMount{
						Name:      pulp.Name + "-signing-scripts",
						MountPath: settings.SigningScriptPath + script.Key,
						SubPath:   script.Key,
						ReadOnly:  true,
					}
					volumeMounts = append(volumeMounts, signingSecretMount)
				}
				break
			}
		}
	}

	if pulpcoreType == settings.API && pulp.Spec.ContainerTokenSecret != "" {
		containerTokenSecretMount := []corev1.VolumeMount{
			{
				Name:      pulp.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_private_key.pem",
				SubPath:   "container_auth_private_key.pem",
				ReadOnly:  true,
			},
			{
				Name:      pulp.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_public_key.pem",
				SubPath:   "container_auth_public_key.pem",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, containerTokenSecretMount...)
	}
	d.volumeMounts = append([]corev1.VolumeMount(nil), volumeMounts...)
}

// setInitContainerVolumeMount defines the init-containers volumes mount points
func (d *CommonDeployment) setInitContainerVolumeMounts(pulp pulpv1.Pulp) {

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
	}

	storageType := GetStorageType(pulp)
	if storageType[0] == SCNameType || storageType[0] == PVCType { // we will mount file-storage if a storageclass or a pvc was provided
		fileStorageMount := corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		}
		volumeMounts = append(volumeMounts, fileStorageMount)
	}
	d.initContainerVolumeMounts = append([]corev1.VolumeMount(nil), volumeMounts...)
}

// setResourceRequirements defines the container resources
func (d *CommonDeployment) setResourceRequirements(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	d.resourceRequirements = reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("ResourceRequirements").Interface().(corev1.ResourceRequirements)
}

// setInitContainerResourceRequirements defines the init-container resources
func (d *CommonDeployment) setInitContainerResourceRequirements(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	d.initContainerResourceRequirements = reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("InitContainer").FieldByName("ResourceRequirements").Interface().(corev1.ResourceRequirements)
}

// setReadinessProbe defines the container readinessprobe
func (d *CommonDeployment) setReadinessProbe(resources any, pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	readinessProbe := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("ReadinessProbe").Interface().(*corev1.Probe)
	ctx := resources.(FunctionResources).Context
	switch pulpcoreType {
	case settings.API:
		if readinessProbe == nil {
			readinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/usr/bin/readyz.py",
							GetAPIRoot(ctx, resources.(FunctionResources).Client, &pulp) + "api/v3/status/",
						},
					},
				},
				FailureThreshold:    1,
				InitialDelaySeconds: 3,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      10,
			}
		}
	case settings.CONTENT:
		if readinessProbe == nil {
			readinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/usr/bin/readyz.py",
							GetContentPathPrefix(ctx, resources.(FunctionResources).Client, &pulp),
						},
					},
				},
				FailureThreshold:    1,
				InitialDelaySeconds: 3,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      10,
			}
		}
	case settings.WORKER:
		if readinessProbe == nil {
			readinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/usr/bin/wait_on_postgres.py",
						},
					},
				},
				FailureThreshold:    1,
				InitialDelaySeconds: 3,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      10,
			}
		}
	}

	d.readinessProbe = readinessProbe
}

// setReadinessProbe defines the container livenessprobe
func (d *CommonDeployment) setLivenessProbe(resources any, pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	livenessProbe := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("LivenessProbe").Interface().(*corev1.Probe)
	ctx := resources.(FunctionResources).Context
	switch pulpcoreType {
	case settings.API:
		if livenessProbe == nil {
			livenessProbe = &corev1.Probe{
				FailureThreshold: 10,
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: GetAPIRoot(ctx, resources.(FunctionResources).Client, &pulp) + "api/v3/status/",
						Port: intstr.IntOrString{
							IntVal: 24817,
						},
						Scheme: corev1.URIScheme("HTTP"),
					},
				},
				InitialDelaySeconds: 3,
				PeriodSeconds:       20,
				SuccessThreshold:    1,
				TimeoutSeconds:      10,
			}
		}
	}
	d.livenessProbe = livenessProbe
}

// setImage defines pulpcore container image
func (d *CommonDeployment) setImage(pulp pulpv1.Pulp) {
	image := os.Getenv("RELATED_IMAGE_PULP")
	if len(pulp.Spec.Image) > 0 && len(pulp.Spec.ImageVersion) > 0 {
		image = pulp.Spec.Image + ":" + pulp.Spec.ImageVersion
	} else if image == "" {
		image = "quay.io/pulp/pulp-minimal:stable"
	}
	d.image = image
}

// setInitContainerImage defines pulpcore init-container image
func (d *CommonDeployment) setInitContainerImage(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	d.initContainerImage = reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("InitContainer").FieldByName("Image").String()
	if len(d.initContainerImage) == 0 {
		d.initContainerImage = d.image
	}
}

// setInitContainers defines initContainers specs
func (d *CommonDeployment) setInitContainers(resources any, pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	args := []string{
		"-c",
		`/usr/bin/wait_on_postgres.py
/usr/bin/wait_on_database_migrations.sh`,
	}
	if pulpcoreType == settings.API {
		args = []string{
			"-c",
			`mkdir -p /var/lib/pulp/{media,assets,tmp}
/usr/bin/wait_on_postgres.py
/usr/bin/wait_on_database_migrations.sh`,
		}
	}

	initContainers := []corev1.Container{
		{
			Name:            "init-container",
			Image:           d.initContainerImage,
			ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
			Env:             d.initContainerEnvVars,
			Command:         []string{"/bin/sh"},
			Args:            args,
			VolumeMounts:    d.initContainerVolumeMounts,
			Resources:       d.initContainerResourceRequirements,
			SecurityContext: SetDefaultSecurityContext(),
		},
	}

	if len(pulp.Spec.SigningSecret) > 0 {
		initContainers = append(initContainers, setGpgInitContainer(resources, pulp))
	}
	d.initContainers = initContainers
}

// setGpgInitContainer returns the definition of a container used to store the gpg keys in the keyring
func setGpgInitContainer(resources any, pulp pulpv1.Pulp) corev1.Container {
	ctx := resources.(FunctionResources).Context
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "gpg-keys",
			MountPath: "/etc/pulp/keys/signing_service.gpg",
			SubPath:   "signing_service.gpg",
			ReadOnly:  true,
		},
	}

	storageType := GetStorageType(pulp)
	if storageType[0] == SCNameType || storageType[0] == PVCType {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		})
	} else {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "ephemeral-gpg",
			MountPath: "/var/lib/pulp/.gnupg",
		})
	}

	// resource requirements
	resourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	signingKeyFingerprint, _ := GetSigningKeyFingerprint(ctx, resources.(FunctionResources).Client, pulp.Spec.SigningSecret, pulp.Namespace)

	// env vars
	envVars := []corev1.EnvVar{{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: signingKeyFingerprint}}
	envVars = append(envVars, corev1.EnvVar{Name: "HOME", Value: "/var/lib/pulp"})

	args := []string{
		`gpg --batch --import /etc/pulp/keys/signing_service.gpg
echo "${PULP_SIGNING_KEY_FINGERPRINT}:6" | gpg --import-ownertrust
`,
	}

	image := pulp.Spec.SigningJob.PulpContainer.Image
	if len(image) == 0 {
		image = pulp.Spec.Image + ":" + pulp.Spec.ImageVersion
	}

	return corev1.Container{
		Name:            "gpg-config",
		Image:           image,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Env:             envVars,
		Command:         []string{"/bin/sh", "-c"},
		Args:            args,
		Resources:       resourceRequirements,
		VolumeMounts:    volumeMounts,
		SecurityContext: SetDefaultSecurityContext(),
	}
}

func pulpcoreApiContainerArgs(pulp pulpv1.Pulp) []string {
	gunicornBindAddress := "[::]:24817"
	if Ipv6Disabled(pulp) {
		gunicornBindAddress = "0.0.0.0:24817"
	}
	return []string{
		"-c",
		`if which pulpcore-api
then
  PULP_API_ENTRYPOINT=("pulpcore-api")
else
  PULP_API_ENTRYPOINT=("gunicorn" "pulpcore.app.wsgi:application" "--name" "pulp-api")
fi
exec "${PULP_API_ENTRYPOINT[@]}" \
--bind "` + gunicornBindAddress + `" \
--timeout "${PULP_GUNICORN_TIMEOUT}" \
--workers "${PULP_API_WORKERS}" \
--access-logformat "${PULP_GUNICORN_ACCESS_LOGFORMAT}" \
--access-logfile -`,
	}
}

func pulpcoreContentContainerArgs(pulp pulpv1.Pulp) []string {
	gunicornBindAddress := "[::]:24816"
	if Ipv6Disabled(pulp) {
		gunicornBindAddress = "0.0.0.0:24816"
	}
	return []string{
		"-c",
		`if which pulpcore-content
then
  PULP_CONTENT_ENTRYPOINT=("pulpcore-content")
else
  PULP_CONTENT_ENTRYPOINT=("gunicorn" "pulpcore.content:server" "--worker-class" "aiohttp.GunicornWebWorker" "--name" "pulp-content")
fi
exec "${PULP_CONTENT_ENTRYPOINT[@]}" \
--bind "` + gunicornBindAddress + `" \
--timeout "${PULP_GUNICORN_TIMEOUT}" \
--workers "${PULP_CONTENT_WORKERS}" \
--access-logformat "${PULP_GUNICORN_ACCESS_LOGFORMAT}" \
--access-logfile -
`,
	}
}

// setContainers defines pulpcore containers specs
func (d *CommonDeployment) setContainers(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	securityContext := SetDefaultSecurityContext()
	var containers []corev1.Container
	switch pulpcoreType {
	case settings.API:
		containers = []corev1.Container{
			{
				Name:            "api",
				Image:           d.image,
				ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
				Command:         []string{"/bin/sh"},
				Args:            pulpcoreApiContainerArgs(pulp),
				Env:             d.envVars,
				Ports: []corev1.ContainerPort{{
					ContainerPort: 24817,
					Protocol:      "TCP",
				}},
				LivenessProbe:   d.livenessProbe,
				ReadinessProbe:  d.readinessProbe,
				Resources:       d.resourceRequirements,
				VolumeMounts:    d.volumeMounts,
				SecurityContext: securityContext,
			},
		}
	case settings.CONTENT:
		containers = []corev1.Container{{
			Name:            "content",
			Image:           d.image,
			ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
			Command:         []string{"/bin/sh"},
			Args:            pulpcoreContentContainerArgs(pulp),
			Resources:       d.resourceRequirements,
			Env:             d.envVars,
			Ports: []corev1.ContainerPort{{
				ContainerPort: 24816,
				Protocol:      "TCP",
			}},
			LivenessProbe:   d.livenessProbe,
			ReadinessProbe:  d.readinessProbe,
			VolumeMounts:    d.volumeMounts,
			SecurityContext: securityContext,
		}}
	case settings.WORKER:
		containers = []corev1.Container{{
			Name:            "worker",
			Image:           d.image,
			ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
			Command:         []string{"/usr/bin/pulp-worker"},
			Env:             d.envVars,
			LivenessProbe:   d.livenessProbe,
			ReadinessProbe:  d.readinessProbe,
			VolumeMounts:    d.volumeMounts,
			Resources:       d.resourceRequirements,
			SecurityContext: securityContext,
		}}
	}
	d.containers = append([]corev1.Container(nil), containers...)
}

// setAnnotations defines the list of pods and deployments annotations
func (d *CommonDeployment) setAnnotations(pulp pulpv1.Pulp, pulpcoreType settings.PulpcoreType) {
	d.podAnnotations = map[string]string{
		"kubectl.kubernetes.io/default-container": strings.ToLower(string(pulpcoreType)),
	}

	if pulp.Status.LastDeploymentUpdate != "" {
		d.podAnnotations["repo-manager.pulpproject.org/restartedAt"] = pulp.Status.LastDeploymentUpdate
	}

	deploymentAnnotations := map[string]string{}
	specField := reflect.ValueOf(pulp.Spec).FieldByName(string(pulpcoreType)).FieldByName("DeploymentAnnotations").Interface().(map[string]string)
	if specField != nil {
		deploymentAnnotations = specField
	}
	// set standard annotations that cannot be overridden by users
	deploymentAnnotations["email"] = "pulp-dev@redhat.com"
	deploymentAnnotations["ignore-check.kube-linter.io/no-node-affinity"] = "Do not check node affinity"

	d.deploymentAnnotations = deploymentAnnotations
}

// setRestartPolicy defines the pod restart policy
func (d *CommonDeployment) setRestartPolicy() {
	d.restartPolicy = corev1.RestartPolicy("Always")
}

// setTerminationPeriod defines the pod terminationGracePeriodSeconds
func (d *CommonDeployment) setTerminationPeriod() {
	terminationPeriod := int64(30)
	d.terminationPeriod = &terminationPeriod
}

// setDnsPolicy defines the pod DNS policy
func (d *CommonDeployment) setDnsPolicy() {
	d.dnsPolicy = corev1.DNSPolicy("ClusterFirst")
}

// setSchedulerName defines the pod schedulername to defaults cheduler
func (d *CommonDeployment) setSchedulerName() {
	d.schedulerName = corev1.DefaultSchedulerName
}

// setTelemetryConfig defines the containers and volumes configuration if telemetry is enabled
func (d *CommonDeployment) setTelemetryConfig(resources any, pulpcoreType settings.PulpcoreType) {
	d.containers, d.volumes = telemetryConfig(resources, d.envVars, d.containers, d.volumes, pulpcoreType)
}

// AddHashLabel creates a label with the calculated hash from the mutated deployment
func AddHashLabel(r FunctionResources, deployment *appsv1.Deployment) {
	var hash string

	// if the object does not exist yet we need to mutate the object to get the
	// default values (I think they are added by the admission controller)
	if err := r.Create(r.Context, deployment, client.DryRunAll); err != nil {
		hash = HashFromMutated(deployment, r)
	} else {
		// When HPA is enabled, exclude replicas from hash calculation
		// to avoid race condition between operator and HPA
		if isHPAManagedDeployment(deployment.Name, r.Pulp) {
			depCopy := deployment.DeepCopy()
			depCopy.Spec.Replicas = nil
			hash = CalculateHash(depCopy.Spec)
		} else {
			hash = CalculateHash(deployment.Spec)
		}
	}

	SetHashLabel(hash, deployment)
}

func (d *CommonDeployment) setLDAPConfigs(resources any) {
	pulp := resources.(FunctionResources).Pulp
	if len(pulp.Spec.LDAP.CA) == 0 {
		return
	}

	ctx := resources.(FunctionResources).Context
	client := resources.(FunctionResources).Client

	// add the CA Secret as a volume
	volumeName := "ldap-cert"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: pulp.Spec.LDAP.CA,
				Items: []corev1.KeyToPath{{
					Key:  "ca.crt",
					Path: "ca.crt",
				}},
			},
		},
	}
	d.volumes = append(d.volumes, volume)

	// retrieve the cert mountPoint from LDAP config Secret
	secretName := pulp.Spec.LDAP.Config
	secret := &corev1.Secret{}
	client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, secret)
	mountPoint := string(secret.Data["auth_ldap_ca_file"])

	// mount the CA Secret
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPoint,
		SubPath:   "ca.crt",
		ReadOnly:  true,
	}
	d.volumeMounts = append(d.volumeMounts, volumeMount)
}

// build constructs the fields used in the deployment specification
func (d *CommonDeployment) build(resources any, pulpcoreType settings.PulpcoreType) {
	pulp := resources.(FunctionResources).Pulp
	d.setReplicas(*pulp, pulpcoreType)
	d.setEnvVars(resources, pulpcoreType)
	d.setStrategy(*pulp, pulpcoreType)
	d.setLabels(*pulp, pulpcoreType)
	d.setAnnotations(*pulp, pulpcoreType)
	d.setAffinity(*pulp, pulpcoreType)
	d.setPodSecurityContext()
	d.setNodeSelector(*pulp, pulpcoreType)
	d.setTolerations(*pulp, pulpcoreType)
	d.setVolumes(resources, pulpcoreType)
	d.setVolumeMounts(*pulp, pulpcoreType)
	d.setResourceRequirements(*pulp, pulpcoreType)
	d.setLivenessProbe(resources, *pulp, pulpcoreType)
	d.setReadinessProbe(resources, *pulp, pulpcoreType)
	d.setImage(*pulp)
	d.setTopologySpreadConstraints(*pulp, pulpcoreType)
	d.setInitContainerResourceRequirements(*pulp, pulpcoreType)
	d.setInitContainerImage(*pulp, pulpcoreType)
	d.setInitContainerVolumeMounts(*pulp)
	d.setInitContainerEnvVars(resources, pulpcoreType)
	d.setLDAPConfigs(resources)
	d.setInitContainers(resources, *pulp, pulpcoreType)
	d.setContainers(*pulp, pulpcoreType)
	d.setRestartPolicy()
	d.setTerminationPeriod()
	d.setDnsPolicy()
	d.setSchedulerName()
	d.setTelemetryConfig(resources, pulpcoreType)
}
