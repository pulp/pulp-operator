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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
)

// ContentResource has the definition and function to provision content objects
type ContentResource struct {
	Definition ResourceDefinition
	Function   func(FunctionResources) client.Object
}

func (r *RepoManagerReconciler) pulpContentController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	var err error
	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Content-Ready"

	// list of pulp-content resources that should be provisioned
	resources := []ContentResource{
		// pulp-content deployment
		{ResourceDefinition{ctx, &appsv1.Deployment{}, pulp.Name + "-content", "Content", conditionType, pulp}, deploymentForPulpContent},
		// pulp-content-svc service
		{ResourceDefinition{ctx, &corev1.Service{}, pulp.Name + "-content-svc", "Content", conditionType, pulp}, serviceForContent},
	}

	// create pulp-content resources
	for _, resource := range resources {
		requeue, err := r.createPulpResource(resource.Definition, resource.Function)
		if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Reconcile Deployment
	deployment := &appsv1.Deployment{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content", Namespace: pulp.Namespace}, deployment)
	expected := deploymentForPulpContent(FunctionResources{ctx, pulp, log, r})
	if deploymentModified(expected.(*appsv1.Deployment), deployment) {
		log.Info("The Content Deployment has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "UpdatingContentDeployment", "Reconciling "+pulp.Name+"-content deployment resource")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling content deployment")
		err = r.Update(ctx, expected.(*appsv1.Deployment))
		if err != nil {
			log.Error(err, "Error trying to update the Content Deployment object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingContentDeployment", "Failed to reconcile "+pulp.Name+"-content deployment resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile content deployment")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Content deployment reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	// Reconcile Service
	cntSvc := &corev1.Service{}
	r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content-svc", Namespace: pulp.Namespace}, cntSvc)
	newCntSvc := serviceForContent(FunctionResources{ctx, pulp, log, r})
	if !equality.Semantic.DeepDerivative(newCntSvc.(*corev1.Service).Spec, cntSvc.Spec) {
		log.Info("The Content Service has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "UpdatingContentService", "Reconciling "+pulp.Name+"-content-svc service")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling content service")
		err = r.Update(ctx, newCntSvc.(*corev1.Service))
		if err != nil {
			log.Error(err, "Error trying to update the Content Service object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingContentService", "Failed to reconcile "+pulp.Name+"-content-svc service: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile content service")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Content service reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpContent returns a pulp-content Deployment object
func deploymentForPulpContent(resources FunctionResources) client.Object {

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
	podSecurityContext := &corev1.PodSecurityContext{}
	IsOpenShift, _ := controllers.IsOpenShift()
	if !IsOpenShift {
		podSecurityContext = &corev1.PodSecurityContext{
			RunAsUser: &runAsUser,
			FSGroup:   &fsGroup,
		}
	}

	nodeSelector := map[string]string{}
	if resources.Pulp.Spec.Content.NodeSelector != nil {
		nodeSelector = resources.Pulp.Spec.Content.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if resources.Pulp.Spec.Content.Tolerations != nil {
		toleration = resources.Pulp.Spec.Content.Tolerations
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
	}

	resourceRequirments := resources.Pulp.Spec.Content.ResourceRequirements

	envVars := []corev1.EnvVar{
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(resources.Pulp.Spec.Content.GunicornTimeout)},
		{Name: "PULP_CONTENT_WORKERS", Value: strconv.Itoa(resources.Pulp.Spec.Content.GunicornWorkers)},
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

	// mountCASpec adds the trusted-ca bundle into []volume and []volumeMount if pulp.Spec.TrustedCA is true
	if IsOpenShift {
		volumes, volumeMounts = mountCASpec(resources.Pulp, volumes, volumeMounts)
	}

	Image := os.Getenv("RELATED_IMAGE_PULP")
	if len(resources.Pulp.Spec.Image) > 0 && len(resources.Pulp.Spec.ImageVersion) > 0 {
		Image = resources.Pulp.Spec.Image + ":" + resources.Pulp.Spec.ImageVersion
	} else if Image == "" {
		Image = "quay.io/pulp/pulp-minimal:stable"
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
						Image:           Image,
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

// labelsForPulpContent returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpContent(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-content",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-content-" + m.Name,
		"app.kubernetes.io/component":  "content",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"app":                          "pulp-content",
		"pulp_cr":                      m.Name,
	}
}

// serviceForContent returns a service object for pulp-content
func serviceForContent(resources FunctionResources) client.Object {

	svc := serviceContentObject(resources.Pulp.Name, resources.Pulp.Namespace, resources.Pulp.Spec.DeploymentType)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(resources.Pulp, svc, resources.RepoManagerReconciler.Scheme)
	return svc
}

func serviceContentObject(name, namespace, deployment_type string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-content-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       deployment_type + "-content",
				"app.kubernetes.io/instance":   deployment_type + "-content-" + name,
				"app.kubernetes.io/component":  "content",
				"app.kubernetes.io/part-of":    deployment_type,
				"app.kubernetes.io/managed-by": deployment_type + "-operator",
				"app":                          "pulp-content",
				"pulp_cr":                      name,
			},
		},
		Spec: serviceContentSpec(name, namespace, deployment_type),
	}
}

// content service spec
func serviceContentSpec(name, namespace, deployment_type string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24816}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Name:       "content-24816",
			Port:       24816,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app.kubernetes.io/name":       deployment_type + "-content",
			"app.kubernetes.io/instance":   deployment_type + "-content-" + name,
			"app.kubernetes.io/component":  "content",
			"app.kubernetes.io/part-of":    deployment_type,
			"app.kubernetes.io/managed-by": deployment_type + "-operator",
			"app":                          "pulp-content",
			"pulp_cr":                      name,
		},
		SessionAffinity:          serviceAffinity,
		Type:                     serviceType,
		PublishNotReadyAddresses: true,
	}
}
