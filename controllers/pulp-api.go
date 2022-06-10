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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
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

	// Ensure the deployment size is the same as the spec
	size := pulp.Spec.Api.Replicas
	if *found.Spec.Replicas != size {
		log.Info("Reconciling Pulp API Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Pulp API Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
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

	// Create pulp-db-fields-encryption secret
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

	runAsUser := int64(0)
	fsGroup := int64(0)

	ls := labelsForPulpApi(m.Name)
	replicas := m.Spec.Api.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-api",
			Namespace: m.Namespace,
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
					Containers: []corev1.Container{{
						Image: "quay.io/pulp/pulp",
						Name:  "api",
						Args:  []string{"pulp-api"},
						Env: []corev1.EnvVar{
							{
								Name:  "POSTGRES_SERVICE_HOST",
								Value: "test-database-svc.pulp-operator-go-system.svc.cluster.local",
							},
							{
								Name:  "POSTGRES_SERVICE_PORT",
								Value: "5432",
							},
							{
								Name:  "PULP_GUNICORN_TIMEOUT",
								Value: "60",
							},
							{
								Name:  "PULP_API_WORKERS",
								Value: "1",
							},
						},
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
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
						FSGroup:   &fsGroup,
					},
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
				},
			},
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForPulpApi returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpApi(name string) map[string]string {
	return map[string]string{"app": "pulp-api", "pulp_cr": name}
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
