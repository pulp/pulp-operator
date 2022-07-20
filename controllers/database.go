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
	"path/filepath"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	"github.com/go-logr/logr"
)

func (r *PulpReconciler) databaseController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// Create pulp-postgres-configuration secret
	pgConfigSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-postgres-configuration", Namespace: pulp.Namespace}, pgConfigSecret)

	expected_secret := databaseConfigSecret(pulp)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new pulp-postgres-configuration secret", "Secret.Namespace", expected_secret.Namespace, "Secret.Name", expected_secret.Name)
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_secret, r.Scheme)
		err = r.Create(ctx, expected_secret)
		if err != nil {
			log.Error(err, "Failed to create new pulp-postgres-configuration secret secret", "Secret.Namespace", expected_secret.Namespace, "Secret.Name", expected_secret.Name)
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-postgres-configuration secret")
		return ctrl.Result{}, err
	}

	// StatefulSet
	pgSts := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database", Namespace: pulp.Namespace}, pgSts)
	expected_sts := statefulSetForDatabase(pulp)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Database StatefulSet", "StatefulSet.Namespace", pgSts.Namespace, "StatefulSet.Name", pgSts.Name)
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_sts, r.Scheme)
		err = r.Create(ctx, expected_sts)
		if err != nil {
			log.Error(err, "Failed to create new Database StatefulSet", "StatefulSet.Namespace", expected_sts.Namespace, "StatefulSet.Name", expected_sts.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database StatefulSet")
		return ctrl.Result{}, err
	}

	// Reconcile StatefulSet
	if !equality.Semantic.DeepDerivative(expected_sts.Spec, pgSts.Spec) {
		log.Info("The Database StatefulSet has been modified! Reconciling ...")
		// Set Pulp instance as the owner and controller
		// not sure if this is the best way to do this, but every time that
		// a reconciliation occurred the object lost the owner reference
		ctrl.SetControllerReference(pulp, expected_sts, r.Scheme)
		err = r.Update(ctx, expected_sts)
		if err != nil {
			log.Error(err, "Error trying to update the Database StatefulSet object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// SERVICE
	dbSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-database-svc", Namespace: pulp.Namespace}, dbSvc)
	expected_svc := serviceForDatabase(pulp)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Database Service", "Service.Namespace", expected_svc.Namespace, "Service.Name", expected_svc.Name)
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_svc, r.Scheme)
		err = r.Create(ctx, expected_svc)
		if err != nil {
			log.Error(err, "Failed to create new Database Service", "Service.Namespace", expected_svc.Namespace, "Service.Name", expected_svc.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if !equality.Semantic.DeepDerivative(expected_svc.Spec, dbSvc.Spec) {
		log.Info("The Database service has been modified! Reconciling ...")
		ctrl.SetControllerReference(pulp, expected_svc, r.Scheme)
		err = r.Update(ctx, expected_svc)
		if err != nil {
			log.Error(err, "Error trying to update the Database Service object ... ")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// statefulSetForDatabase returns a postgresql Deployment object
func statefulSetForDatabase(m *repomanagerv1alpha1.Pulp) *appsv1.StatefulSet {
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

	postgresDataPath := ""
	if m.Spec.Database.PostgresDataPath == "" {
		postgresDataPath = "/var/lib/postgresql/data/pgdata"
	} else {
		postgresDataPath = m.Spec.Database.PostgresDataPath
	}

	postgresInitdbArgs := ""
	if m.Spec.Database.PostgresInitdbArgs == "" {
		postgresInitdbArgs = "--auth-host=scram-sha-256"
	} else {
		postgresInitdbArgs = m.Spec.Database.PostgresInitdbArgs
	}

	postgresHostAuthMethod := ""
	if m.Spec.Database.PostgresHostAuthMethod == "" {
		postgresHostAuthMethod = "scram-sha-256"
	} else {
		postgresHostAuthMethod = m.Spec.Database.PostgresHostAuthMethod
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
		{Name: "PGDATA", Value: postgresDataPath},
		{Name: "POSTGRES_INITDB_ARGS", Value: postgresInitdbArgs},
		{Name: "POSTGRES_HOST_AUTH_METHOD", Value: postgresHostAuthMethod},
	}

	postgresStorageSize := resource.Quantity{}
	if reflect.DeepEqual(m.Spec.Database.PostgresStorageRequirements, resource.Quantity{}) {
		postgresStorageSize = resource.MustParse("8Gi")
	} else {
		postgresStorageSize = m.Spec.Database.PostgresStorageRequirements
	}

	pvcSpec := corev1.PersistentVolumeClaimSpec{}
	if m.Spec.Database.PostgresStorageClass != nil {
		pvcSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): postgresStorageSize,
				},
			},
			StorageClassName: m.Spec.Database.PostgresStorageClass,
		}
	} else {
		pvcSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): postgresStorageSize,
				},
			},
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
			MountPath: filepath.Dir(postgresDataPath),
			SubPath:   filepath.Base(postgresDataPath),
		},
	}

	resources := m.Spec.Database.ResourceRequirements

	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"pg_isready -U " + m.Spec.DeploymentType + " -h 127.0.0.1 -p 5432",
				},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}

	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh",
					"-i",
					"-c",
					"pg_isready -U " + m.Spec.DeploymentType + " -h 127.0.0.1 -p 5432",
				},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    6,
		SuccessThreshold:    1,
	}

	postgresImage := ""
	if m.Spec.Database.PostgresImage == "" {
		postgresImage = "postgres:13"
	} else {
		postgresImage = m.Spec.Database.PostgresImage
	}

	containerPort := int32(0)
	if m.Spec.Database.PostgresPort == 0 {
		containerPort = int32(5432)
	} else {
		containerPort = int32(m.Spec.Database.PostgresPort)
	}

	return &appsv1.StatefulSet{
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
						Image: postgresImage,
						Name:  "postgres",
						Args:  args,
						Env:   envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: containerPort,
							Name:          "postgres",
						}},
						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						VolumeMounts:   volumeMounts,
						Resources:      resources,
					}},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplate,
		},
	}
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
func serviceForDatabase(m *repomanagerv1alpha1.Pulp) *corev1.Service {
	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 5432}
	serviceType := corev1.ServiceType("ClusterIP")

	return &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-database-svc",
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
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
				"pulp_cr": m.Name,
			},
			SessionAffinity: serviceAffinity,
			Type:            serviceType,
		},
	}
}

// pulp-postgres-configuration secret
func databaseConfigSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {

	sslMode := ""
	if m.Spec.Database.PostgresSSLMode == "" {
		sslMode = "prefer"
	} else {
		sslMode = m.Spec.Database.PostgresSSLMode
	}

	return &corev1.Secret{
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
			"sslmode":  sslMode,
			"type":     "managed",
		},
	}

}
