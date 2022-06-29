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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PulpReconciler) pulpApiController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	pgUser, _ := r.retrieveSecretData(ctx, "username", pulp.Name+"-postgres-configuration", pulp.Namespace, log)
	pgPwd, _ := r.retrieveSecretData(ctx, "password", pulp.Name+"-postgres-configuration", pulp.Namespace, log)

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForPulpApi(pulp)
		log.Info("Creating a new Pulp API Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Pulp API Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp API Deployment")
		return ctrl.Result{}, err
	}

	// https://github.com/kubernetes-sigs/kubebuilder/issues/592
	// this is always being considered NOT equal because []containers and []volumes
	// are always "pointing" to different places. Example:
	/*
		3c3
		< [(*"k8s.io/api/core/v1.Volume")(0xc000140f00),(*"k8s.io/api/core/v1.Volume")(0xc000140ff8),...
		---
		> [(*"k8s.io/api/core/v1.Volume")(0xc000140a00),(*"k8s.io/api/core/v1.Volume")(0xc000140af8),...
		9c9
		< [(*"k8s.io/api/core/v1.Container")(0xc00037c2c0)]
		---
		> [(*"k8s.io/api/core/v1.Container")(0xc000157a20)]
	*/

	// Ensure the deployment template spec is as expected
	// https://github.com/kubernetes-sigs/kubebuilder/issues/592
	expected_deployment_spec := deploymentSpec(pulp)
	if !equality.Semantic.DeepDerivative(expected_deployment_spec.Template.Spec, found.Spec.Template.Spec) {
		log.Info("The API deployment has been modified! Reconciling ...")
		found.Spec.Template.Spec = expected_deployment_spec.Template.Spec
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Error trying to update the API deployment object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// pending reconcile number o api replicas and deployment labels

	/*
		// as a workaround to check if pulp the deployment is in sync with Pulp CR instance
		// we can compare each field managed by Pulp CR and how they are in current deployment
		updated, modified := r.checkDeployment(pulp, found)
		if modified {
			log.Info("The API deployment has been modified! Reconciling ...")
			patch := client.MergeFrom(found.DeepCopy())
			err = r.Patch(ctx, updated, patch)
			if err != nil {
				log.Error(err, "Error trying to update the API deployment object ... ")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
	*/

	// Create pulp-server secret
	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-server", Namespace: pulp.Namespace}, secret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		sec := r.pulpServerSecret(pulp, string(pgUser), string(pgPwd))
		log.Info("Creating a new pulp-server secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
		err = r.Create(ctx, sec)
		if err != nil {
			log.Error(err, "Failed to create new pulp-server secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-server secret")
		return ctrl.Result{}, err
	}

	// Create pulp-db-fields-encryption secret
	dbFieldsEnc := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-db-fields-encryption", Namespace: pulp.Namespace}, dbFieldsEnc)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		dbFields := r.pulpDBFieldsEncryptionSecret(pulp)
		log.Info("Creating a new pulp-db-fields-encryption secret", "Secret.Namespace", dbFields.Namespace, "Secret.Name", dbFields.Name)
		err = r.Create(ctx, dbFields)
		if err != nil {
			log.Error(err, "Failed to create new pulp-db-fields-encryption secret", "Secret.Namespace", dbFields.Namespace, "Secret.Name", dbFields.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-db-fields-encryption secret")
		return ctrl.Result{}, err
	}

	// Create pulp-admin-password secret
	adminSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-admin-password", Namespace: pulp.Namespace}, adminSecret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		adminPwd := r.pulpAdminPasswordSecret(pulp)
		log.Info("Creating a new pulp-admin-password secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
		err = r.Create(ctx, adminPwd)
		if err != nil {
			log.Error(err, "Failed to create new pulp-admin-password secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-admin-password secret")
		return ctrl.Result{}, err
	}

	// SERVICE
	apiSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api-svc", Namespace: pulp.Namespace}, apiSvc)

	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		newApiSvc := r.serviceForAPI(pulp)
		log.Info("Creating a new API Service", "Service.Namespace", newApiSvc.Namespace, "Service.Name", newApiSvc.Name)
		err = r.Create(ctx, newApiSvc)
		if err != nil {
			log.Error(err, "Failed to create new API Service", "Service.Namespace", newApiSvc.Namespace, "Service.Name", newApiSvc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get API Service")
		return ctrl.Result{}, err
	}

	// Ensure the service spec is as expected
	expected_api_spec := serviceAPISpec(pulp.Name)

	if !reflect.DeepEqual(expected_api_spec, apiSvc.Spec) {
		log.Info("The API service has been modified! Reconciling ...")
		err = r.Update(ctx, serviceAPIObject(pulp.Name, pulp.Namespace))
		if err != nil {
			log.Error(err, "Error trying to update the API Service object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpApi returns a pulp-api Deployment object
func (r *PulpReconciler) deploymentForPulpApi(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	// deployment definition
	dep := deploymentObject(m)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// deploymentObject allows to reuse the deployment object in
// - the deploymentForPulpApi (deployment provision) method and
// - the reconciliation (client.Write.Update) step
func deploymentObject(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-api",
			Namespace: m.Namespace,
			Annotations: map[string]string{
				"email": "pulp-dev@redhat.com",
				"ignore-check.kube-linter.io/no-node-affinity": "Do not check node affinity",
			},
			Labels: map[string]string{
				"app.kubernetes.io/name":       m.Spec.DeploymentType + "-api",
				"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-api-" + m.Name,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
				"owner":                        "pulp-dev",
			},
		},
		Spec: deploymentSpec(m),
	}
}

// deploymentSpec containns only the .spec definition
func deploymentSpec(m *repomanagerv1alpha1.Pulp) appsv1.DeploymentSpec {
	replicas := m.Spec.Api.Replicas
	ls := labelsForPulpApi(m)

	affinity := &corev1.Affinity{}
	if m.Spec.Api.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Api.Affinity.NodeAffinity
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
		nodeSelector = m.Spec.Api.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if m.Spec.Api.Tolerations != nil {
		toleration = m.Spec.Api.Tolerations
	}

	topologySpreadConstraint := []corev1.TopologySpreadConstraint{}
	if m.Spec.Api.TopologySpreadConstraints != nil {
		topologySpreadConstraint = m.Spec.Api.TopologySpreadConstraints
	}

	envVars := []corev1.EnvVar{
		{Name: "POSTGRES_SERVICE_HOST", Value: m.Name + "-database-svc." + m.Namespace + ".svc"},
		{Name: "POSTGRES_SERVICE_PORT", Value: strconv.Itoa(m.Spec.Database.PostgresPort)},
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(m.Spec.Api.GunicornTimeout)},
		{Name: "PULP_API_WORKERS", Value: strconv.Itoa(m.Spec.Api.GunicornWorkers)},
	}

	redisEnvVars := []corev1.EnvVar{}
	if m.Spec.CacheEnabled {
		redisEnvVars = []corev1.EnvVar{
			{Name: "REDIS_SERVICE_HOST", Value: m.Name + "-redis-svc"},
			{Name: "REDIS_SERVICE_PORT", Value: strconv.Itoa(m.Spec.RedisPort)},
		}
	}

	envVars = append(envVars, redisEnvVars...)

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
			Name: m.Name + "-admin-password",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.Name + "-admin-password",
					Items: []corev1.KeyToPath{{
						Path: "admin-password",
						Key:  "password",
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

	if m.Spec.ObjectStorageS3Secret == "" {
		fileStorage := corev1.Volume{
			Name: "file-storage",
			VolumeSource: corev1.VolumeSource{
				// Configuring as EmptyDir for test purposes
				EmptyDir: &corev1.EmptyDirVolumeSource{},
				// PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				// ClaimName: m.Name + "-file-storage",
				// },
			},
		}
		volumes = append(volumes, fileStorage)
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

	if m.Spec.ContainerTokenSecret != "" {
		containerTokenSecretVolume := corev1.Volume{
			Name: m.Name + "-container-auth-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.Spec.ContainerTokenSecret,
					Items: []corev1.KeyToPath{
						{
							Key:  "container_auth_public_key.pem",
							Path: m.Spec.ContainerAuthPublicKey,
						},
						{
							Key:  "container_auth_private_key.pem",
							Path: m.Spec.ContainerAuthPrivateKey,
						},
					},
				},
			},
		}
		volumes = append(volumes, containerTokenSecretVolume)
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      m.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      m.Name + "-admin-password",
			MountPath: "/etc/pulp/pulp-admin-password",
			SubPath:   "admin-password",
			ReadOnly:  true,
		},
		{
			Name:      m.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      "tmp-file-storage",
			MountPath: "/var/lib/pulp/tmp",
		},
		{
			Name:      "assets-file-storage",
			MountPath: "/var/lib/pulp/assets",
		},
	}

	if m.Spec.ObjectStorageS3Secret == "" {
		fileStorageMount := corev1.VolumeMount{
			Name:      "file-storage",
			ReadOnly:  false,
			MountPath: "/var/lib/pulp",
		}
		volumeMounts = append(volumeMounts, fileStorageMount)
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

	if m.Spec.ContainerTokenSecret != "" {
		containerTokenSecretMount := []corev1.VolumeMount{
			{
				Name:      m.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_private_key.pem",
				SubPath:   "container_auth_private_key.pem",
				ReadOnly:  true,
			},
			{
				Name:      m.Name + "-container-auth-certs",
				MountPath: "/etc/pulp/keys/container_auth_public_key.pem",
				SubPath:   "container_auth_pulblic_key.pem",
				ReadOnly:  true,
			},
		}
		volumeMounts = append(volumeMounts, containerTokenSecretMount...)
	}

	resources := m.Spec.Api.ResourceRequirements

	// the following variables are defined to avoid issues with reconciliation
	restartPolicy := corev1.RestartPolicy("Always")
	terminationGracePeriodSeconds := int64(30)
	dnsPolicy := corev1.DNSPolicy("ClusterFirst")
	schedulerName := corev1.DefaultSchedulerName

	return appsv1.DeploymentSpec{
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
				ServiceAccountName:        m.Spec.DeploymentType + "-operator-sa",
				TopologySpreadConstraints: topologySpreadConstraint,
				Containers: []corev1.Container{{
					Name:            "api",
					Image:           m.Spec.Image + ":" + m.Spec.ImageVersion,
					ImagePullPolicy: corev1.PullPolicy(m.Spec.ImagePullPolicy),
					Args:            []string{"pulp-api"},
					Env:             envVars,
					Ports: []corev1.ContainerPort{{
						ContainerPort: 24817,
						Protocol:      "TCP",
					}},
					/* WIP
					LivenessProbe:  &corev1.Probe{},
					ReadinessProbe: &corev1.Probe{}, */
					Resources:    resources,
					VolumeMounts: volumeMounts,
				}},

				/* the following configs are not defined on pulp-operator (ansible version)  */
				/* but i'll keep it here just in case we can manage to make deepequal usable */
				RestartPolicy:                 restartPolicy,
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				DNSPolicy:                     dnsPolicy,
				SchedulerName:                 schedulerName,
			},
		},
	}

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

func (r *PulpReconciler) pulpServerSecret(m *repomanagerv1alpha1.Pulp, pgUser, pgPwd string) *corev1.Secret {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-server",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"settings.py": `
CONTENT_ORIGIN = "http://` + m.Name + `-content-svc.` + m.Namespace + `.svc.cluster.local:24816"
API_ROOT = "/pulp/"
CACHE_ENABLED = "False"
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys"
DATABASES = { 'default' : { 'HOST': '` + m.Name + `-database-svc.` + m.Namespace + `.svc.cluster.local', 'ENGINE': 'django.db.backends.postgresql_psycopg2', 'NAME': 'pulp', 'USER': '` + pgUser + `', 'PASSWORD': '` + pgPwd + `', 'CONN_MAX_AGE': 0, 'PORT': '5432'}}
`,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

// pulp-db-fields-encryption secret
func (r *PulpReconciler) pulpDBFieldsEncryptionSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-db-fields-encryption",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"database_fields.symmetric.key": "81HqDtbqAywKSOumSha3BhWNOdQ26slT6K0YaZeZyPs=",
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

// pulp-admin-passowrd
func (r *PulpReconciler) pulpAdminPasswordSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-admin-password",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"password": "password",
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

/*
// PROBABLY DEPRECATED IN FAVOR OF equality.Semantic.DeepDerivative (need to do more tests)
// verify if deployment matches the definition on Pulp CR
func (r *PulpReconciler) checkDeployment(m *repomanagerv1alpha1.Pulp, current *appsv1.Deployment) (*appsv1.Deployment, bool) {

	modified := false
	patch := current.DeepCopy()

	// check replicas
	if !reflect.DeepEqual(*current.Spec.Replicas, m.Spec.Api.Replicas) {
		*patch.Spec.Replicas = m.Spec.Api.Replicas
		modified = true
	}

	// check affinity
	if !reflect.DeepEqual(current.Spec.Template.Spec.Affinity.NodeAffinity, m.Spec.Api.Affinity.NodeAffinity) {
		patch.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: m.Spec.Api.Affinity.NodeAffinity,
		}
		modified = true
	}

	// check security context (is_k8s)
	runAsUser := int64(0)
	fsGroup := int64(0)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
		FSGroup:   &fsGroup,
	}
	if m.Spec.IsK8s && (!reflect.DeepEqual(current.Spec.Template.Spec.SecurityContext.RunAsUser, podSecurityContext.RunAsUser) ||
		!reflect.DeepEqual(current.Spec.Template.Spec.SecurityContext.FSGroup, podSecurityContext.FSGroup)) {
		patch.Spec.Template.Spec.SecurityContext = podSecurityContext
		modified = true
	}
	if !m.Spec.IsK8s && (current.Spec.Template.Spec.SecurityContext.RunAsUser != nil ||
		current.Spec.Template.Spec.SecurityContext.FSGroup != nil) {
		patch.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}
		modified = true
	}

	// check nodeSelector
	if !reflect.DeepEqual(current.Spec.Template.Spec.NodeSelector, m.Spec.Api.NodeSelector) {
		patch.Spec.Template.Spec.NodeSelector = m.Spec.Api.NodeSelector
		modified = true
	}

	// check tolerations
	if !reflect.DeepEqual(current.Spec.Template.Spec.Tolerations, m.Spec.Api.Tolerations) {
		patch.Spec.Template.Spec.Tolerations = m.Spec.Api.Tolerations
		modified = true
	}

	// check volumes [PENDING]
	// check serviceAccount [PENDING]

	// check topologySpreadConstraints
	if !reflect.DeepEqual(current.Spec.Template.Spec.TopologySpreadConstraints, m.Spec.Api.TopologySpreadConstraints) {
		patch.Spec.Template.Spec.TopologySpreadConstraints = m.Spec.Api.TopologySpreadConstraints
		modified = true
	}

	// check container image [PENDING]
	// check container imagePullPolicy [PENDING]
	// check container env [PENDING]
	// check container livenessProbe [PENDING]
	// check container readinessProbe [PENDING]
	// check container resources [PENDING]
	// check container mounts [PENDING]

	return patch, modified
}
*/

// serviceForAPI returns a service object for pulp-api
func (r *PulpReconciler) serviceForAPI(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceAPIObject(m.Name, m.Namespace)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceAPIObject(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-api-svc",
			Namespace: namespace,
		},
		Spec: serviceAPISpec(name),
	}
}

// api service spec
func serviceAPISpec(name string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 24817}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		ClusterIP:             "None",
		ClusterIPs:            []string{"None"},
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Port:       24817,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app":     "pulp-api",
			"pulp_cr": name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}
