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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *PulpReconciler) pulpApiController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {
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
	/*
		// Ensure the deployment template spec is as expected
		expected_deployment_spec := deploymentSpec(pulp)
		if !reflect.DeepEqual(expected_deployment_spec.Template.Spec, found.Spec.Template.Spec) {
			log.Info("The API deployment has been modified! Reconciling ...")
			err = r.Update(ctx, deploymentObject(pulp))
			if err != nil {
				log.Error(err, "Error trying to update the API deployment object ... ")
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true}, nil
		}
	*/

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

	// Create pulp-server secret
	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-server", Namespace: pulp.Namespace}, secret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		sec := r.pulpServerSecret(pulp)
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
		log.Info("Creating a new pulp-admin-secret secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
		err = r.Create(ctx, adminPwd)
		if err != nil {
			log.Error(err, "Failed to create new pulp-admin-secret secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-admin-secret secret")
		return ctrl.Result{}, err
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
		{Name: "POSTGRES_SERVICE_HOST", Value: m.Name + "-postgres-" + m.Spec.PostgresVersion},
		{Name: "POSTGRES_SERVICE_PORT", Value: strconv.Itoa(m.Spec.PostgresPort)},
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
				NodeSelector:              nodeSelector,
				Tolerations:               toleration,
				SecurityContext:           podSecurityContext,
				TopologySpreadConstraints: topologySpreadConstraint,
				Containers: []corev1.Container{{
					Image: "quay.io/pulp/pulp",
					Name:  "api",
					Args:  []string{"pulp-api"},
					Env:   envVars,
					Ports: []corev1.ContainerPort{{
						ContainerPort: 24817,
						Protocol:      "TCP",
					}},
					VolumeMounts: []corev1.VolumeMount{
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
					},
				}},
				Volumes: []corev1.Volume{
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
								SecretName: m.Name + "-db-fields-encryption",
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
				},
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
	}
}

func (r *PulpReconciler) pulpServerSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-server",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"settings.py": `
CONTENT_ORIGIN = "http://test-pulp-content-svc.pulp-operator-go-system.svc.cluster.local:24816"
API_ROOT = "/pulp/"
CACHE_ENABLED = "False"
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys"
DATABASES = { 'default' : { 'HOST': 'test-database-svc.pulp-operator-go-system.svc.cluster.local', 'ENGINE': 'django.db.backends.postgresql_psycopg2', 'NAME': 'pulp', 'USER': 'admin', 'PASSWORD': 'password', 'CONN_MAX_AGE': 0, 'PORT': '5432'}}
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

// verify if deployment matches the definition on Pulp CR
func (r *PulpReconciler) checkDeployment(m *repomanagerv1alpha1.Pulp, current *appsv1.Deployment) (*appsv1.Deployment, bool) {

	modified := false
	patch := current.DeepCopy()
	if !reflect.DeepEqual(*current.Spec.Replicas, m.Spec.Api.Replicas) {
		*patch.Spec.Replicas = m.Spec.Api.Replicas
		modified = true
	}

	if !reflect.DeepEqual(current.Spec.Template.Spec.Affinity.NodeAffinity, m.Spec.Api.Affinity.NodeAffinity) {
		patch.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: m.Spec.Api.Affinity.NodeAffinity,
		}
		modified = true
	}

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

	if !reflect.DeepEqual(current.Spec.Template.Spec.NodeSelector, m.Spec.Api.NodeSelector) {
		patch.Spec.Template.Spec.NodeSelector = m.Spec.Api.NodeSelector
		modified = true
	}

	if !reflect.DeepEqual(current.Spec.Template.Spec.Tolerations, m.Spec.Api.Tolerations) {
		patch.Spec.Template.Spec.Tolerations = m.Spec.Api.Tolerations
		modified = true
	}

	if !reflect.DeepEqual(current.Spec.Template.Spec.TopologySpreadConstraints, m.Spec.Api.TopologySpreadConstraints) {
		patch.Spec.Template.Spec.TopologySpreadConstraints = m.Spec.Api.TopologySpreadConstraints
		modified = true
	}

	return patch, modified
}
