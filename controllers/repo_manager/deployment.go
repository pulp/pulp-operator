package repo_manager

import (
	"os"
	"strconv"

	"github.com/pulp/pulp-operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentType int

const (
	API_DEPLOYMENT deploymentType = iota
	CONTENT_DEPLOYMENT
	WORKER_DEPLOYMENT
)

// DeploymentObj represents the k8s "Deployment" resource
type DeploymentObj struct {
	// Deployer is the abstraction for the different pulp deployment types (api,content,worker)
	Deployer
}

// initDeployment returns a deployment object of type "deployer" based on k8s distribution and
// Pulp deployment type (api,worker or content)
func initDeployment(dt deploymentType) *DeploymentObj {

	isOpenshift, _ := controllers.IsOpenShift()

	switch dt {
	case API_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{DeploymentAPIOCP{}}
		}
		return &DeploymentObj{DeploymentAPIVanilla{}}
	case WORKER_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{DeploymentWorkerOCP{}}
		}
		return &DeploymentObj{DeploymentWorkerVanilla{}}
	case CONTENT_DEPLOYMENT:
		if isOpenshift {
			return &DeploymentObj{DeploymentContentOCP{}}
		}
		return &DeploymentObj{DeploymentContentVanilla{}}
	}

	return &DeploymentObj{}
}

// Deployer is an interface for the several deployment types:
// - api deployment in vanilla k8s or ocp
// - content deployment in vanilla k8s or ocp
// - worker deployment in vanilla k8s or ocp
type Deployer interface {
	deploy(FunctionResources) client.Object
}

type DeploymentAPIVanilla struct{}

// deploy returns a pulp-api Deployment object
func (DeploymentAPIVanilla) deploy(resources FunctionResources) client.Object {
	replicas := resources.Pulp.Spec.Api.Replicas
	ls := labelsForPulpApi(resources.Pulp)

	affinity := &corev1.Affinity{}
	if resources.Pulp.Spec.Api.Affinity != nil {
		affinity = resources.Pulp.Spec.Api.Affinity
	}

	if resources.Pulp.Spec.Affinity != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		affinity.NodeAffinity = resources.Pulp.Spec.Affinity.NodeAffinity
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := resources.Pulp.Spec.Api.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	runAsUser := int64(700)
	fsGroup := int64(700)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
		FSGroup:   &fsGroup,
	}

	nodeSelector := map[string]string{}
	if resources.Pulp.Spec.Api.NodeSelector != nil {
		nodeSelector = resources.Pulp.Spec.Api.NodeSelector
	} else if resources.Pulp.Spec.NodeSelector != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		nodeSelector = resources.Pulp.Spec.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if resources.Pulp.Spec.Api.Tolerations != nil {
		toleration = resources.Pulp.Spec.Api.Tolerations
	} else if resources.Pulp.Spec.Tolerations != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		toleration = resources.Pulp.Spec.Tolerations
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if resources.Pulp.Spec.Api.TopologySpreadConstraints != nil {
		topologySpreadConstraint = resources.Pulp.Spec.Api.TopologySpreadConstraints
	} else if resources.Pulp.Spec.TopologySpreadConstraints != nil {
		topologySpreadConstraint = resources.Pulp.Spec.TopologySpreadConstraints
	}

	gunicornWorkers := strconv.Itoa(resources.Pulp.Spec.Api.GunicornWorkers)
	if resources.Pulp.Spec.GunicornAPIWorkers > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		gunicornWorkers = strconv.Itoa(resources.Pulp.Spec.GunicornAPIWorkers)
	}
	gunicornTimeout := strconv.Itoa(resources.Pulp.Spec.Api.GunicornTimeout)
	if resources.Pulp.Spec.GunicornTimeout > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		gunicornTimeout = strconv.Itoa(resources.Pulp.Spec.GunicornTimeout)
	}
	envVars := []corev1.EnvVar{
		{Name: "PULP_GUNICORN_TIMEOUT", Value: gunicornTimeout},
		{Name: "PULP_API_WORKERS", Value: gunicornWorkers},
	}

	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(resources.Pulp.Spec.PostgresConfigurationSecret) > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		postgresEnvVars := []corev1.EnvVar{
			{
				Name: "POSTGRES_SERVICE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	} else if len(resources.Pulp.Spec.Database.ExternalDBSecret) == 0 {
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
				SubPath:   "container_auth_public_key.pem",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, containerTokenSecretMount...)
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
	image := os.Getenv("RELATED_IMAGE_PULP")
	if len(resources.Pulp.Spec.Image) > 0 && len(resources.Pulp.Spec.ImageVersion) > 0 {
		image = resources.Pulp.Spec.Image + ":" + resources.Pulp.Spec.ImageVersion
	} else if image == "" {
		image = "quay.io/pulp/pulp-minimal:stable"
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
						Image:           image,
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

type DeploymentContentVanilla struct{}

// deploy returns a pulp-content Deployment object
func (DeploymentContentVanilla) deploy(resources FunctionResources) client.Object {

	labels := map[string]string{
		"app.kubernetes.io/name":       resources.Pulp.Spec.DeploymentType + "-content",
		"app.kubernetes.io/instance":   resources.Pulp.Spec.DeploymentType + "-content-" + resources.Pulp.Name,
		"app.kubernetes.io/component":  "content",
		"app.kubernetes.io/part-of":    resources.Pulp.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": resources.Pulp.Spec.DeploymentType + "-operator",
		"app":                          "pulp-content",
		"pulp_cr":                      resources.Pulp.Name,
		"owner":                        "pulp-dev",
	}

	replicas := resources.Pulp.Spec.Content.Replicas
	ls := labelsForPulpContent(resources.Pulp)

	affinity := &corev1.Affinity{}
	if resources.Pulp.Spec.Content.Affinity != nil {
		affinity = resources.Pulp.Spec.Content.Affinity
	}

	if resources.Pulp.Spec.Affinity != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		affinity.NodeAffinity = resources.Pulp.Spec.Affinity.NodeAffinity
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := resources.Pulp.Spec.Content.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	// pulp image is built to run with user 0
	// we are enforcing the containers to run as 1000
	runAsUser := int64(700)
	fsGroup := int64(700)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
		FSGroup:   &fsGroup,
	}

	nodeSelector := map[string]string{}
	if resources.Pulp.Spec.Content.NodeSelector != nil {
		nodeSelector = resources.Pulp.Spec.Content.NodeSelector
	} else if resources.Pulp.Spec.NodeSelector != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		nodeSelector = resources.Pulp.Spec.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if resources.Pulp.Spec.Content.Tolerations != nil {
		toleration = resources.Pulp.Spec.Content.Tolerations
	} else if resources.Pulp.Spec.Tolerations != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		toleration = resources.Pulp.Spec.Tolerations
	}

	dbFieldsEncryptionSecret := ""
	if resources.Pulp.Spec.DBFieldsEncryptionSecret == "" {
		dbFieldsEncryptionSecret = resources.Pulp.Name + "-db-fields-encryption"
	} else {
		dbFieldsEncryptionSecret = resources.Pulp.Spec.DBFieldsEncryptionSecret
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
		}
		volumes = append(volumes, emptyDir...)
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if resources.Pulp.Spec.Content.TopologySpreadConstraints != nil {
		topologySpreadConstraint = resources.Pulp.Spec.Content.TopologySpreadConstraints
	} else if resources.Pulp.Spec.TopologySpreadConstraints != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		topologySpreadConstraint = resources.Pulp.Spec.TopologySpreadConstraints
	}

	resourceRequirments := resources.Pulp.Spec.Content.ResourceRequirements

	gunicornWorkers := strconv.Itoa(resources.Pulp.Spec.Content.GunicornWorkers)
	if resources.Pulp.Spec.GunicornContentWorkers > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		gunicornWorkers = strconv.Itoa(resources.Pulp.Spec.GunicornContentWorkers)
	}
	gunicornTimeout := strconv.Itoa(resources.Pulp.Spec.Content.GunicornTimeout)
	if resources.Pulp.Spec.GunicornTimeout > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		gunicornTimeout = strconv.Itoa(resources.Pulp.Spec.GunicornTimeout)
	}
	envVars := []corev1.EnvVar{
		{Name: "PULP_GUNICORN_TIMEOUT", Value: gunicornTimeout},
		{Name: "PULP_CONTENT_WORKERS", Value: gunicornWorkers},
	}

	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(resources.Pulp.Spec.PostgresConfigurationSecret) > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		postgresEnvVars := []corev1.EnvVar{
			{
				Name: "POSTGRES_SERVICE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	} else if len(resources.Pulp.Spec.Database.ExternalDBSecret) == 0 {
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
			{Name: "SIGNING_SERVICE", Value: "ansible-default"},
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      resources.Pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      resources.Pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
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

	image := os.Getenv("RELATED_IMAGE_PULP")
	if len(resources.Pulp.Spec.Image) > 0 && len(resources.Pulp.Spec.ImageVersion) > 0 {
		image = resources.Pulp.Spec.Image + ":" + resources.Pulp.Spec.ImageVersion
	} else if image == "" {
		image = "quay.io/pulp/pulp-minimal:stable"
	}

	readinessProbe := resources.Pulp.Spec.Content.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/usr/bin/readyz.py",
						getPulpSetting(resources.Pulp, "content_path_prefix"),
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
	livenessProbe := resources.Pulp.Spec.Content.LivenessProbe

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-content",
			Namespace: resources.Pulp.Namespace,
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
					ServiceAccountName:        resources.Pulp.Name,
					TopologySpreadConstraints: topologySpreadConstraint,
					Containers: []corev1.Container{{
						Name:            "content",
						Image:           image,
						ImagePullPolicy: corev1.PullPolicy(resources.Pulp.Spec.ImagePullPolicy),
						Args:            []string{"pulp-content"},
						Resources:       resourceRequirments,
						Env:             envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 24816,
							Protocol:      "TCP",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						VolumeMounts:   volumeMounts,
					}},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, dep, resources.RepoManagerReconciler.Scheme)
	return dep
}

type DeploymentWorkerVanilla struct{}

// deploy returns a pulp-worker Deployment object
func (DeploymentWorkerVanilla) deploy(resources FunctionResources) client.Object {
	ls := labelsForPulpWorker(resources.Pulp)
	labels := map[string]string{
		"app.kubernetes.io/name":       resources.Pulp.Spec.DeploymentType + "-worker",
		"app.kubernetes.io/instance":   resources.Pulp.Spec.DeploymentType + "-worker-" + resources.Pulp.Name,
		"app.kubernetes.io/component":  "worker",
		"app.kubernetes.io/part-of":    resources.Pulp.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": resources.Pulp.Spec.DeploymentType + "-operator",
		"owner":                        "pulp-dev",
	}
	replicas := resources.Pulp.Spec.Worker.Replicas

	affinity := &corev1.Affinity{}
	if resources.Pulp.Spec.Worker.Affinity != nil {
		affinity = resources.Pulp.Spec.Worker.Affinity
	}

	if resources.Pulp.Spec.Affinity != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		affinity.NodeAffinity = resources.Pulp.Spec.Affinity.NodeAffinity
	}

	// if no strategy is defined in pulp CR we are setting `strategy.Type` with the
	// default value ("RollingUpdate"), this will be helpful during the reconciliation
	// when a strategy was previously defined and eventually the field is removed
	strategy := resources.Pulp.Spec.Worker.Strategy
	if strategy.Type == "" {
		strategy.Type = "RollingUpdate"
	}

	// pulp image is built to run with user 0
	// we are enforcing the containers to run as 1000
	runAsUser := int64(700)
	fsGroup := int64(700)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
		FSGroup:   &fsGroup,
	}

	nodeSelector := map[string]string{}
	if resources.Pulp.Spec.Worker.NodeSelector != nil {
		nodeSelector = resources.Pulp.Spec.Worker.NodeSelector
	} else if resources.Pulp.Spec.NodeSelector != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		nodeSelector = resources.Pulp.Spec.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if resources.Pulp.Spec.Worker.Tolerations != nil {
		toleration = resources.Pulp.Spec.Worker.Tolerations
	} else if resources.Pulp.Spec.Tolerations != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		toleration = resources.Pulp.Spec.Tolerations
	}

	dbFieldsEncryptionSecret := ""
	if resources.Pulp.Spec.DBFieldsEncryptionSecret == "" {
		dbFieldsEncryptionSecret = resources.Pulp.Name + "-db-fields-encryption"
	} else {
		dbFieldsEncryptionSecret = resources.Pulp.Spec.DBFieldsEncryptionSecret
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
		{
			Name: resources.Pulp.Name + "-ansible-tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
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
		}
		volumes = append(volumes, emptyDir...)
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if resources.Pulp.Spec.Worker.TopologySpreadConstraints != nil {
		topologySpreadConstraint = resources.Pulp.Spec.Worker.TopologySpreadConstraints
	} else if resources.Pulp.Spec.TopologySpreadConstraints != nil { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		topologySpreadConstraint = resources.Pulp.Spec.TopologySpreadConstraints
	}

	envVars := []corev1.EnvVar{}
	var dbHost, dbPort string

	// if there is no ExternalDBSecret defined, we should
	// use the postgres instance provided by the operator
	if len(resources.Pulp.Spec.PostgresConfigurationSecret) > 0 { // [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
		postgresEnvVars := []corev1.EnvVar{
			{
				Name: "POSTGRES_SERVICE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_HOST",
					},
				},
			}, {
				Name: "POSTGRES_SERVICE_PORT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: resources.Pulp.Spec.PostgresConfigurationSecret,
						},
						Key: "POSTGRES_PORT",
					},
				},
			},
		}
		envVars = append(envVars, postgresEnvVars...)
	} else if len(resources.Pulp.Spec.Database.ExternalDBSecret) == 0 {
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
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      resources.Pulp.Name + "-ansible-tmp",
			MountPath: "/.ansible/tmp",
		},
		{
			Name:      resources.Pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      resources.Pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
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

	resourceRequirements := resources.Pulp.Spec.Worker.ResourceRequirements
	image := os.Getenv("RELATED_IMAGE_PULP")
	if len(resources.Pulp.Spec.Image) > 0 && len(resources.Pulp.Spec.ImageVersion) > 0 {
		image = resources.Pulp.Spec.Image + ":" + resources.Pulp.Spec.ImageVersion
	} else if image == "" {
		image = "quay.io/pulp/pulp-minimal:stable"
	}

	readinessProbe := resources.Pulp.Spec.Worker.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/usr/bin/wait_on_postgres.py",
					},
				},
			},
			FailureThreshold:    10,
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		}
	}
	livenessProbe := resources.Pulp.Spec.Worker.LivenessProbe

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.Pulp.Name + "-worker",
			Namespace: resources.Pulp.Namespace,
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
					ServiceAccountName:        resources.Pulp.Name,
					TopologySpreadConstraints: topologySpreadConstraint,
					Containers: []corev1.Container{{
						Name:            "worker",
						Image:           image,
						ImagePullPolicy: corev1.PullPolicy(resources.Pulp.Spec.ImagePullPolicy),
						Args:            []string{"pulp-worker"},
						Env:             envVars,
						LivenessProbe:   livenessProbe,
						ReadinessProbe:  readinessProbe,
						VolumeMounts:    volumeMounts,
						Resources:       resourceRequirements,
					}},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, dep, resources.RepoManagerReconciler.Scheme)
	return dep
}
