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

func (r *PulpReconciler) pulpWorkerController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {
	workerDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-worker", Namespace: pulp.Namespace}, workerDeployment)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		newWorkerDeployment := r.deploymentForPulpWorker(pulp)
		log.Info("Creating a new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
		err = r.Create(ctx, newWorkerDeployment)
		if err != nil {
			log.Error(err, "Failed to create new Pulp Worker Deployment", "Deployment.Namespace", newWorkerDeployment.Namespace, "Deployment.Name", newWorkerDeployment.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp Worker Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	workerReplicas := pulp.Spec.Worker.Replicas
	if *workerDeployment.Spec.Replicas != workerReplicas {
		log.Info("Reconciling Pulp Worker Deployment", "Deployment.Namespace", workerDeployment.Namespace, "Deployment.Name", workerDeployment.Name)
		workerDeployment.Spec.Replicas = &workerReplicas
		err = r.Update(ctx, workerDeployment)
		if err != nil {
			log.Error(err, "Failed to update Pulp Worker Deployment", "Deployment.Namespace", workerDeployment.Namespace, "Deployment.Name", workerDeployment.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForPulpWorker returns a pulp-worker Deployment object
func (r *PulpReconciler) deploymentForPulpWorker(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	runAsUser := int64(0)
	fsGroup := int64(0)

	ls := labelsForPulpWorker(m.Name)
	replicas := m.Spec.Worker.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-worker",
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
						Name:  "worker",
						Args:  []string{"pulp-worker"},
						Env: []corev1.EnvVar{
							{
								Name:  "POSTGRES_SERVICE_HOST",
								Value: m.Name + "-database-svc." + m.Namespace + ".svc",
							},
							{
								Name:  "POSTGRES_SERVICE_PORT",
								Value: "5432",
							},
							{
								Name:  "PULP_GUNICORN_TIMEOUT",
								Value: "60",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
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
							{
								Name:      "file-storage-tmp",
								MountPath: "/var/lib/pulp/tmp",
								ReadOnly:  false,
							},
						},
					}},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &runAsUser,
						FSGroup:   &fsGroup,
					},
					Volumes: []corev1.Volume{
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
							Name: "file-storage-tmp",
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

// labelsForPulpWorker returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForPulpWorker(name string) map[string]string {
	return map[string]string{"app": "pulp-worker", "pulp_cr": name}
}
