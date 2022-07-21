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
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) pulpContentController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// Controller Deployment
	cntDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content", Namespace: pulp.Namespace}, cntDeployment)
	newCntDeployment := r.deploymentForPulpContent(pulp)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pulp Content Deployment", "Deployment.Namespace", newCntDeployment.Namespace, "Deployment.Name", newCntDeployment.Name)
		err = r.Create(ctx, newCntDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Content Deployment", "Deployment.Namespace", newCntDeployment.Namespace, "Deployment.Name", newCntDeployment.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Content Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment
	if !equality.Semantic.DeepDerivative(newCntDeployment.Spec, cntDeployment.Spec) {
		log.Info("The Content Deployment has been modified! Reconciling ...")
		err = r.Update(ctx, newCntDeployment)
		if err != nil {
			log.Error(err, "Error trying to update the Content Deployment object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// SERVICE
	cntSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-content-svc", Namespace: pulp.Namespace}, cntSvc)
	newCntSvc := r.serviceForContent(pulp)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		log.Info("Creating a new Content Service", "Service.Namespace", newCntSvc.Namespace, "Service.Name", newCntSvc.Name)
		err = r.Create(ctx, newCntSvc)
		if err != nil {
			log.Error(err, "Failed to create new Content Service", "Service.Namespace", newCntSvc.Namespace, "Service.Name", newCntSvc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Content Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if !equality.Semantic.DeepDerivative(newCntSvc.Spec, cntSvc.Spec) {
		log.Info("The Content Service has been modified! Reconciling ...")
		err = r.Update(ctx, newCntSvc)
		if err != nil {
			log.Error(err, "Error trying to update the Content Service object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpContent returns a pulp-content Deployment object
func (r *PulpReconciler) deploymentForPulpContent(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	labels := map[string]string{
		"app.kubernetes.io/name":       m.Spec.DeploymentType + "-content",
		"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-content-" + m.Name,
		"app.kubernetes.io/component":  "content",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"owner":                        "pulp-dev",
	}

	replicas := m.Spec.Content.Replicas
	ls := labelsForPulpContent(m)

	affinity := &corev1.Affinity{}
	if m.Spec.Api.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Content.Affinity.NodeAffinity
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
	if m.Spec.Api.NodeSelector != nil {
		nodeSelector = m.Spec.Content.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if m.Spec.Api.Tolerations != nil {
		toleration = m.Spec.Content.Tolerations
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
	if m.Spec.Api.TopologySpreadConstraints != nil {
		topologySpreadConstraint = m.Spec.Api.TopologySpreadConstraints
	}

	resources := m.Spec.Content.ResourceRequirements

	containerPort := 0
	if m.Spec.Database.PostgresPort == 0 {
		containerPort = 5432
	} else {
		containerPort = m.Spec.Database.PostgresPort
	}

	envVars := []corev1.EnvVar{
		{Name: "POSTGRES_SERVICE_HOST", Value: m.Name + "-database-svc." + m.Namespace + ".svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: strconv.Itoa(containerPort)},
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(m.Spec.Content.GunicornTimeout)},
		{Name: "PULP_CONTENT_WORKERS", Value: strconv.Itoa(m.Spec.Content.GunicornWorkers)},
	}
	if m.Spec.CacheEnabled {
		redisEnvVars := []corev1.EnvVar{
			{Name: "REDIS_SERVICE_HOST", Value: m.Name + "-redis-svc." + m.Namespace},
			{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(m.Spec.RedisPort)},
		}
		envVars = append(envVars, redisEnvVars...)
	}
	if m.Spec.SigningSecret != "" {
		// for now, we are just dumping the error, but we should handle it
		signingKeyFingerprint, _ := r.getSigningKeyFingerprint(m.Spec.SigningSecret, m.Namespace)
		signingKeyEnvVars := []corev1.EnvVar{
			{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: signingKeyFingerprint},
			{Name: "SIGNING_SERVICE", Value: "ansible-default"},
		}
		envVars = append(envVars, signingKeyEnvVars...)
	}

	volumeMounts := []corev1.VolumeMount{
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

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-content",
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
						Name:            "content",
						Image:           m.Spec.Image + ":" + m.Spec.ImageVersion,
						ImagePullPolicy: corev1.PullPolicy(m.Spec.ImagePullPolicy),
						Args:            []string{"pulp-content"},
						Resources:       resources,
						Env:             envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 24816,
							Protocol:      "TCP",
						}},
						// LivenessProbe:  livenessProbe,
						// ReadinessProbe: readinessProbe,
						VolumeMounts: volumeMounts,
					}},
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
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
func (r *PulpReconciler) serviceForContent(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceContentObject(m.Name, m.Namespace)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceContentObject(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-content-svc",
			Namespace: namespace,
		},
		Spec: serviceContentSpec(name),
	}
}

// content service spec
func serviceContentSpec(name string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24816}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		ClusterIP:             "None",
		ClusterIPs:            []string{"None"},
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Port:       24816,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app":     "pulp-content",
			"pulp_cr": name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}
