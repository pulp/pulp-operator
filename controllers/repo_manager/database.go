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
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *RepoManagerReconciler) databaseController(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp, log logr.Logger) (ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := cases.Title(language.English, cases.Compact).String(pulp.Spec.DeploymentType) + "-Database-Ready"

	secretName := settings.DefaultDBSecret(pulp.Name)
	// Create pulp-postgres-configuration secret
	pgConfigSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, pgConfigSecret)
	expected_secret := databaseConfigSecret(pulp)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new "+secretName+" Secret", "Secret.Namespace", expected_secret.Namespace, "Secret.Name", secretName)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingDatabasePostgresSecret", "Creating "+secretName+" Secret resource")
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_secret, r.Scheme)
		err = r.Create(ctx, expected_secret)
		if err != nil {
			log.Error(err, "Failed to create "+secretName+" Secret", "Secret.Namespace", expected_secret.Namespace, "Secret.Name", secretName)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingDatabasePostgresSecret", "Failed to create "+secretName+" Secret resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create "+secretName+" Secret")
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", secretName+" Secret created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get "+secretName+" Secret")
		return ctrl.Result{}, err
	}

	// StatefulSet
	statefulSetName := settings.DefaultDBStatefulSet(pulp.Name)
	pgSts := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: statefulSetName, Namespace: pulp.Namespace}, pgSts)
	expected_sts := statefulSetForDatabase(pulp)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Database StatefulSet", "StatefulSet.Namespace", pgSts.Namespace, "StatefulSet.Name", statefulSetName)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingDatabaseSts", "Creating "+statefulSetName+" StatefulSet resource")
		controllers.CheckEmptyDir(pulp, controllers.DatabaseResource)
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_sts, r.Scheme)
		err = r.Create(ctx, expected_sts)
		if err != nil {
			log.Error(err, "Failed to create new Database StatefulSet", "StatefulSet.Namespace", expected_sts.Namespace, "StatefulSet.Name", statefulSetName)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingDatabaseSts", "Failed to create "+statefulSetName+" Statefulset resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create database StatefulSet")
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Database StatefulSet created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database StatefulSet")
		return ctrl.Result{}, err
	}

	// Reconcile StatefulSet
	if !equality.Semantic.DeepDerivative(expected_sts.Spec, pgSts.Spec) {
		log.Info("The " + statefulSetName + " StatefulSet has been modified! Reconciling ...")
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "UpdatingDatabaseSts", "Reconciling "+statefulSetName+" Statefulset resource")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling "+statefulSetName+" StatefulSet")
		// Set Pulp instance as the owner and controller
		// not sure if this is the best way to do this, but every time that
		// a reconciliation occurred the object lost the owner reference
		ctrl.SetControllerReference(pulp, expected_sts, r.Scheme)
		err = r.Update(ctx, expected_sts)
		if err != nil {
			log.Error(err, "Error trying to update the "+statefulSetName+" StatefulSet object ... ")
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingDatabaseSts", "Failed to reconcile "+statefulSetName+" Statefulset resource")
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile "+statefulSetName+" StatefulSet")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", statefulSetName+" StatefulSet reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, nil
	}

	// SERVICE
	svcName := settings.DBService(pulp.Name)
	dbSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: pulp.Namespace}, dbSvc)
	expected_svc := serviceForDatabase(pulp)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Database Service", "Service.Namespace", expected_svc.Namespace, "Service.Name", svcName)
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "CreatingDatabaseService", "Creating "+svcName+" Service resource")
		// Set Pulp instance as the owner and controller
		ctrl.SetControllerReference(pulp, expected_svc, r.Scheme)
		err = r.Create(ctx, expected_svc)
		if err != nil {
			log.Error(err, "Failed to create new Database Service", "Service.Namespace", expected_svc.Namespace, "Service.Name", svcName)
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorCreatingDatabaseService", "Failed to create "+svcName+" Service resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to create database service")
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Created", "Database service created")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Database Service")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if !equality.Semantic.DeepDerivative(expected_svc.Spec, dbSvc.Spec) {
		log.Info("The Database service has been modified! Reconciling ...")
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "UpdatingDatabaseService", "Reconciling "+svcName+" Service resource")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updating", "Reconciling database service")
		ctrl.SetControllerReference(pulp, expected_svc, r.Scheme)
		err = r.Update(ctx, expected_svc)
		if err != nil {
			log.Error(err, "Error trying to update the Database Service object ... ")
			controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionFalse, conditionType, "ErrorUpdatingDatabaseService", "Failed to reconcile "+svcName+" Service resource: "+err.Error())
			r.recorder.Event(pulp, corev1.EventTypeWarning, "Failed", "Failed to reconcile database service")
			return ctrl.Result{}, err
		}
		r.recorder.Event(pulp, corev1.EventTypeNormal, "Updated", "Database service reconciled")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
	}

	// we should only update the status when Database-Ready==false
	if v1.IsStatusConditionFalse(pulp.Status.Conditions, conditionType) {
		controllers.UpdateStatus(ctx, r.Client, pulp, metav1.ConditionTrue, conditionType, "DatabaseTasksFinished", "All Database tasks ran successfully")
		r.recorder.Event(pulp, corev1.EventTypeNormal, "DatabaseReady", "All Database tasks ran successfully")
	}

	return ctrl.Result{}, nil
}

// statefulSetForDatabase returns a postgresql Deployment object
func statefulSetForDatabase(m *repomanagerpulpprojectorgv1beta2.Pulp) *appsv1.StatefulSet {

	ls := labelsForDatabase(m)
	//replicas := m.Spec.Database.Replicas
	replicas := int32(1)

	affinity := &corev1.Affinity{}
	if m.Spec.Database.Affinity != nil {
		affinity = m.Spec.Database.Affinity
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
	if m.Spec.Database.PostgresDataPath != "" {
		postgresDataPath = m.Spec.Database.PostgresDataPath
	} else {
		postgresDataPath = "/var/lib/postgresql/data/pgdata"
	}

	postgresInitdbArgs := ""
	if m.Spec.Database.PostgresInitdbArgs != "" {
		postgresInitdbArgs = m.Spec.Database.PostgresInitdbArgs
	} else {
		postgresInitdbArgs = "--auth-host=scram-sha-256"
	}

	postgresHostAuthMethod := ""
	if m.Spec.Database.PostgresHostAuthMethod != "" {
		postgresHostAuthMethod = m.Spec.Database.PostgresHostAuthMethod
	} else {
		postgresHostAuthMethod = "scram-sha-256"
	}

	postgresConfigurationSecret := settings.DefaultDBSecret(m.Name)
	envVars := []corev1.EnvVar{
		{
			Name: "POSTGRESQL_DATABASE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: postgresConfigurationSecret,
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
						Name: postgresConfigurationSecret,
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
						Name: postgresConfigurationSecret,
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
						Name: postgresConfigurationSecret,
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
						Name: postgresConfigurationSecret,
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
						Name: postgresConfigurationSecret,
					},
					Key: "password",
				},
			},
		},
		{Name: "PGDATA", Value: postgresDataPath},
		{Name: "POSTGRES_INITDB_ARGS", Value: postgresInitdbArgs},
		{Name: "POSTGRES_HOST_AUTH_METHOD", Value: postgresHostAuthMethod},
	}
	///pvcSpec := corev1.PersistentVolumeClaimSpec{}
	volumeClaimTemplate := []corev1.PersistentVolumeClaim{}
	volumes := []corev1.Volume{}
	_, storageType := controllers.MultiStorageConfigured(m, "Database")

	storageClass := m.Spec.Database.PostgresStorageClass

	volumeName := settings.DefaultDBPVC(m.Name)
	// if SC defined, we should use the PVC claimed by STS
	if storageType[0] == controllers.SCNameType {

		// Temporarily while we don't find a fix for backup and json.Unmarshal issue
		postgresStorageSize := resource.MustParse("8Gi")
		if m.Spec.Database.PostgresStorageRequirements != "" {
			postgresStorageSize = resource.MustParse(m.Spec.Database.PostgresStorageRequirements)
		}
		storageRequirements := corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceName(corev1.ResourceStorage): postgresStorageSize,
			},
		}

		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources:        storageRequirements,
				StorageClassName: storageClass,
			},
		}
		volumeClaimTemplate = append(volumeClaimTemplate, pvc)

		// if .spec.Database.PVC defined we should use the PVC provisioned by user
	} else if storageType[0] == controllers.PVCType {
		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.Spec.Database.PVC,
				},
			},
		}
		volumes = append(volumes, volume)

		// if there is no SC nor PVC nor object storage defined we will mount an emptyDir
	} else if storageType[0] == controllers.EmptyDirType {
		emptyDir := []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		volumes = append(volumes, emptyDir...)
	}

	pgDataMountPath := filepath.Dir(postgresDataPath)
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: pgDataMountPath,
			SubPath:   filepath.Base(pgDataMountPath),
		},
	}

	resources := m.Spec.Database.ResourceRequirements

	livenessProbe := m.Spec.Database.LivenessProbe
	if livenessProbe == nil {
		livenessProbe = &corev1.Probe{
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
	}

	readinessProbe := m.Spec.Database.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
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
	}

	postgresImage := os.Getenv("RELATED_IMAGE_PULP_POSTGRES")
	if len(m.Spec.Database.PostgresImage) > 0 {
		postgresImage = m.Spec.Database.PostgresImage
	} else if postgresImage == "" {
		postgresImage = "docker.io/library/postgres:13"
	}

	containerPort := int32(0)
	if m.Spec.Database.PostgresPort == 0 {
		containerPort = int32(5432)
	} else {
		containerPort = int32(m.Spec.Database.PostgresPort)
	}

	runAsUser := int64(999)
	fsGroup := int64(999)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.DefaultDBStatefulSet(m.Name),
			Namespace: m.Namespace,
			Labels:    ls,
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
					ServiceAccountName: settings.PulpServiceAccount(m.Name),
					SecurityContext:    &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup},
					Containers: []corev1.Container{{
						Image: postgresImage,
						Name:  "postgres",
						Args:  args,
						Env:   envVars,
						Ports: []corev1.ContainerPort{{
							ContainerPort: containerPort,
							Name:          "postgres",
						}},
						LivenessProbe:   livenessProbe,
						ReadinessProbe:  readinessProbe,
						VolumeMounts:    volumeMounts,
						Resources:       resources,
						SecurityContext: controllers.SetDefaultSecurityContext(),
					}},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: volumeClaimTemplate,
		},
	}
}

// labelsForDatabase returns the labels for selecting the resources
// belonging to the given pulp CR name.
func labelsForDatabase(m *repomanagerpulpprojectorgv1beta2.Pulp) map[string]string {
	return settings.PulpcoreLabels(*m, "database")
}

// serviceForDatabase returns a service object for postgres pods
func serviceForDatabase(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.Service {
	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: 5432}
	serviceType := corev1.ServiceType("ClusterIP")

	return &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.DBService(m.Name),
			Namespace: m.Namespace,
			Labels:    labelsForDatabase(m),
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
			Selector:        labelsForDatabase(m),
			SessionAffinity: serviceAffinity,
			Type:            serviceType,
		},
	}
}

// pulp-postgres-configuration secret
func databaseConfigSecret(m *repomanagerpulpprojectorgv1beta2.Pulp) *corev1.Secret {

	sslMode := ""
	if m.Spec.Database.PostgresSSLMode == "" {
		sslMode = "prefer"
	} else {
		sslMode = m.Spec.Database.PostgresSSLMode
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.DefaultDBSecret(m.Name),
			Namespace: m.Namespace,
			Labels:    settings.CommonLabels(*m),
		},
		StringData: map[string]string{
			"password": createPwd(32),
			"username": m.Spec.DeploymentType,
			"database": m.Spec.DeploymentType,
			"port":     "5432",
			"host":     m.Name + "-database-svc",
			"sslmode":  sslMode,
			"type":     "managed",
		},
	}
}
