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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) databaseController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// Create pulp-postgres-configuration secret
	pgConfigSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-postgres-configuration", Namespace: pulp.Namespace}, pgConfigSecret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		newPgConfigSecret := r.databaseConfigSecret(pulp)
		log.Info("Creating a new pulp-postgres-configuration secret", "Secret.Namespace", newPgConfigSecret.Namespace, "Secret.Name", newPgConfigSecret.Name)
		err = r.Create(ctx, newPgConfigSecret)
		if err != nil {
			log.Error(err, "Failed to create new pulp-postgres-configuration secret secret", "Secret.Namespace", newPgConfigSecret.Namespace, "Secret.Name", newPgConfigSecret.Name)
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-postgres-configuration secret")
		return ctrl.Result{}, err
	}

	// DEPLOYMENT
	found := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database", Namespace: pulp.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		sts := r.statefulSetForDatabase(pulp)
		log.Info("Creating a new Database StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		err = r.Create(ctx, sts)
		if err != nil {
			log.Error(err, "Failed to create new Database StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database StatefulSet")
		return ctrl.Result{}, err
	}

	// SERVICE
	dbSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database-svc", Namespace: pulp.Namespace}, dbSvc)

	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := r.serviceForDatabase(pulp)
		log.Info("Creating a new Database Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Database Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database Service")
		return ctrl.Result{}, err
	}

	// Ensure the service spec is as expected
	expected_spec := serviceDBSpec(pulp.Name)

	if !reflect.DeepEqual(expected_spec, dbSvc.Spec) {
		log.Info("The Database service has been modified! Reconciling ...")
		err = r.Update(ctx, serviceDBObject(pulp.Name, pulp.Namespace))
		if err != nil {
			log.Error(err, "Error trying to update the Database Service object ... ")
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// deploymentForDatabase returns a postgresql Deployment object
func (r *PulpReconciler) statefulSetForDatabase(m *repomanagerv1alpha1.Pulp) *appsv1.StatefulSet {
	ls := labelsForDatabase(m)
	//replicas := m.Spec.Database.Replicas
	replicas := int32(1)

	affinity := &corev1.Affinity{}
	if m.Spec.Database.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Database.Affinity.NodeAffinity
	}

	nodeSelector := map[string]string{}
	if m.Spec.Database.NodeSelector != nil {
		nodeSelector = m.Spec.Database.NodeSelector
	}

	toleration := []corev1.Toleration{}
	if m.Spec.Database.Tolerations != nil {
		toleration = m.Spec.Database.Tolerations
	}

	args := []string{}
	if len(m.Spec.Database.PostgresExtraArgs) > 0 {
		args = m.Spec.Database.PostgresExtraArgs
	}

	envVars := []corev1.EnvVar{
		{
			Name: "POSTGRESQL_DATABASE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "database",
				},
			},
		},
		{
			Name: "POSTGRESQL_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "POSTGRESQL_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "password",
				},
			},
		},
		{
			Name: "POSTGRES_DB",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "database",
				},
			},
		},
		{
			Name: "POSTGRES_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "POSTGRES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.Name + "-postgres-configuration",
					},
					Key: "password",
				},
			},
		},
		{Name: "PGDATA", Value: m.Spec.Database.PostgresDataPath},
		{Name: "POSTGRES_INITDB_ARGS", Value: m.Spec.Database.PostgresInitdbArgs},
		{Name: "POSTGRES_HOST_AUTH_METHOD", Value: m.Spec.Database.PostgresHostAuthMethod},
	}

	/*
		pvcSpec := corev1.PersistentVolumeClaimSpec{}
		if m.Spec.Database.PostgresStorageClass != nil {
			pvcSpec = corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources:        m.Spec.Database.PostgresStorageRequirements,
				StorageClassName: m.Spec.Database.PostgresStorageClass,
			}
		} else {
			pvcSpec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources:   m.Spec.Database.PostgresStorageRequirements,
			}
		}

			volumeClaimTemplate := []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "postgres",
				},
				Spec: pvcSpec,
			},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "postgres",
					MountPath: filepath.Dir(m.Spec.Database.PostgresDataPath),
					SubPath:   filepath.Base(m.Spec.Database.PostgresDataPath),
				},
			}
	*/

	resources := m.Spec.Database.ResourceRequirements

	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-database",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "postgres",
				"app.kubernetes.io/instance":   "postgres-" + m.Name,
				"app.kubernetes.io/component":  "database",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
				"owner":                        "pulp-dev",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Affinity:           affinity,
					NodeSelector:       nodeSelector,
					Tolerations:        toleration,
					ServiceAccountName: "pulp-operator-go-controller-manager",
					Containers: []corev1.Container{{
						Image: m.Spec.Database.PostgresImage,
						Name:  "postgres",
						Args:  args,
						Env:   envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(m.Spec.Database.PostgresPort),
							Name:          "postgres",
						}},
						/* WIP
						LivenessProbe:  &corev1.Probe{},
						ReadinessProbe: &corev1.Probe{},
						VolumeMounts: volumeMounts,  */
						Resources: resources,
					}},
				},
			},
			//VolumeClaimTemplates: volumeClaimTemplate,
		},
	}
	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForDatabase returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForDatabase(m *repomanagerv1alpha1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "postgres",
		"app.kubernetes.io/instance":   "postgres-" + m.Name,
		"app.kubernetes.io/component":  "database",
		"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
		"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
		"owner":                        "pulp-dev",
		"app":                          "postgresql",
		"pulp_cr":                      m.Name,
	}
}

// serviceForDatabase returns a service object for postgres pods
func (r *PulpReconciler) serviceForDatabase(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceDBObject(m.Name, m.Namespace)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceDBObject(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-database-svc",
			Namespace: namespace,
		},
		Spec: serviceDBSpec(name),
	}
}

// database service spec
// [TO-DO]
// - the other services will probably be the same as this one with only other
//   names, ports and selectors. This function could be modified to address all pulp
//   service configurations (maybe just adding the corev1.ServicePort and a dictionary
//   for the selectors as function parameters would be enough).
func serviceDBSpec(name string) corev1.ServiceSpec {

	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 5432}
	serviceType := corev1.ServiceType("ClusterIP")

	return corev1.ServiceSpec{
		ClusterIP:             "None",
		ClusterIPs:            []string{"None"},
		InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
		IPFamilies:            []corev1.IPFamily{"IPv4"},
		IPFamilyPolicy:        &ipFamilyPolicyType,
		Ports: []corev1.ServicePort{{
			Port:       5432,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app":     "postgresql",
			"pulp_cr": name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}

// pulp-postgres-configuration secret
func (r *PulpReconciler) databaseConfigSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-postgres-configuration",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"password": createPwd(32),
			"username": m.Spec.DeploymentType,
			"database": m.Spec.DeploymentType,
			"port":     "5432",
			"host":     m.Name + "-postgres-" + m.Spec.Database.PostgresVersion,
			"sslmode":  m.Spec.Database.PostgresSSLMode,
			"type":     "managed",
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}
