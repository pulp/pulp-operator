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

package pulp

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/go-logr/logr"
	repomanagerv1alpha1 "github.com/pulp/pulp-operator/api/v1alpha1"
	"github.com/pulp/pulp-operator/controllers"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PulpReconciler) pulpApiController(ctx context.Context, pulp *repomanagerv1alpha1.Pulp, log logr.Logger) (ctrl.Result, error) {

	// pulp-file-storage
	if len(pulp.Spec.ObjectStorageAzureSecret) == 0 && len(pulp.Spec.ObjectStorageS3Secret) == 0 {
		pvcFound := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-file-storage", Namespace: pulp.Namespace}, pvcFound)
		expected_pvc := r.fileStoragePVC(pulp)

		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new Pulp API File Storage PVC", "PVC.Namespace", expected_pvc.Namespace, "PVC.Name", expected_pvc.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingPVC", "Creating "+pulp.Name+"-file-storage PVC resource")
			err = r.Create(ctx, expected_pvc)
			if err != nil {
				log.Error(err, "Failed to create new Pulp File Storage PVC", "PVC.Namespace", expected_pvc.Namespace, "PVC.Name", expected_pvc.Name)
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingPVC", "Failed to create "+pulp.Name+"-file-storage PVC: "+err.Error())
				return ctrl.Result{}, err
			}
			// PVC created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get Pulp API File Storage PVC")
			return ctrl.Result{}, err
		}

		// Reconcile PVC
		if !equality.Semantic.DeepDerivative(expected_pvc.Spec, pvcFound.Spec) {
			log.Info("The PVC has been modified! Reconciling ...")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "UpdatingFileStoragePVC", "Reconciling "+pulp.Name+"-file-storage PVC resource")
			err = r.Update(ctx, expected_pvc)
			if err != nil {
				log.Error(err, "Error trying to update the PVC object ... ")
				r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorUpdatingFileStoragePVC", "Failed to reconcile "+pulp.Name+"-file-storage PVC resource")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, nil
		}
	}

	// Create pulp-server secret
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-server", Namespace: pulp.Namespace}, secret)

	// Create pulp-server secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		// retrieve database credentials from postgres-secret only if we are not passing an external database settings
		sec := r.pulpServerSecret(ctx, pulp, log)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingServerSecret", "Creating "+pulp.Name+"-server secret")
		log.Info("Creating a new pulp-server secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
		err = r.Create(ctx, sec)
		if err != nil {
			log.Error(err, "Failed to create new pulp-server secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingServerSecret", "Failed to create "+pulp.Name+"-server secret: "+err.Error())
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
		dbFields := pulpDBFieldsEncryptionSecret(pulp)
		ctrl.SetControllerReference(pulp, dbFields, r.Scheme)
		log.Info("Creating a new pulp-db-fields-encryption secret", "Secret.Namespace", dbFields.Namespace, "Secret.Name", dbFields.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingDBFieldsEncryptionSecret", "Creating "+pulp.Name+"-db-fields-encryption secret")
		err = r.Create(ctx, dbFields)
		if err != nil {
			log.Error(err, "Failed to create new pulp-db-fields-encryption secret", "Secret.Namespace", dbFields.Namespace, "Secret.Name", dbFields.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingDBFieldsEncryptionSecret", "Failed to create "+pulp.Name+"-db-fields-encryption secret: "+err.Error())
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
	adminSecretName := pulp.Name + "-admin-password"
	if len(pulp.Spec.AdminPasswordSecret) > 1 {
		adminSecretName = pulp.Spec.AdminPasswordSecret
	}
	err = r.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: pulp.Namespace}, adminSecret)

	// Create the secret in case it is not found
	if err != nil && errors.IsNotFound(err) {
		adminPwd := pulpAdminPasswordSecret(pulp)
		ctrl.SetControllerReference(pulp, adminPwd, r.Scheme)
		log.Info("Creating a new pulp-admin-password secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingAdminPasswordSecret", "Creating "+pulp.Name+"-admin-password secret")
		err = r.Create(ctx, adminPwd)
		if err != nil {
			log.Error(err, "Failed to create new pulp-admin-password secret", "Secret.Namespace", adminPwd.Namespace, "Secret.Name", adminPwd.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingAdminPasswordSecret", "Failed to create "+pulp.Name+"-admin-password secret: "+err.Error())
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-admin-password secret")
		return ctrl.Result{}, err
	}

	// Create pulp-container-auth secret
	containerAuth := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-container-auth", Namespace: pulp.Namespace}, containerAuth)

	// Create the secret in case it is not found
	if pulp.Spec.ContainerTokenSecret == "" && err != nil && errors.IsNotFound(err) {
		authSecret := pulpContainerAuth(pulp)
		// Following legacy pulp-operator implementation we are not setting ownerReferences to avoid garbage collection
		//ctrl.SetControllerReference(pulp, adminPwd, r.Scheme)
		log.Info("Creating a new pulp-container-auth secret", "Secret.Namespace", authSecret.Namespace, "Secret.Name", authSecret.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingContainerAuthSecret", "Creating "+pulp.Name+"-container-auth secret")
		err = r.Create(ctx, authSecret)
		if err != nil {
			log.Error(err, "Failed to create new pulp-container-auth secret", "Secret.Namespace", authSecret.Namespace, "Secret.Name", authSecret.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingContainerAuthSecret", "Failed to create "+pulp.Name+"-container-auth secret: "+err.Error())
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get pulp-container-auth secret")
		return ctrl.Result{}, err
	}

	// Create pulp-api deployment
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api", Namespace: pulp.Namespace}, found)
	dep := r.deploymentForPulpApi(pulp)

	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		log.Info("Creating a new Pulp API Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingApiDeployment", "Creating "+pulp.Name+"-api deployment")
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Pulp API Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingApiDeployment", "Failed to create "+pulp.Name+"-api deployment: "+err.Error())
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pulp API Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment template spec is as expected
	// https://github.com/kubernetes-sigs/kubebuilder/issues/592
	if !equality.Semantic.DeepDerivative(dep.Spec, found.Spec) {
		log.Info("The API deployment has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "UpdatingApiDeployment", "Reconciling "+pulp.Name+"-api deployment")
		err = r.Update(ctx, dep)
		if err != nil {
			log.Error(err, "Error trying to update the API deployment object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorUpdatingApiDeployment", "Failed to reconcile "+pulp.Name+"-api deployment: "+err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// SERVICE
	apiSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Name + "-api-svc", Namespace: pulp.Namespace}, apiSvc)
	newApiSvc := r.serviceForAPI(pulp)

	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new API Service", "Service.Namespace", newApiSvc.Namespace, "Service.Name", newApiSvc.Name)
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "CreatingApiService", "Creating "+pulp.Name+"-api-svc service")
		err = r.Create(ctx, newApiSvc)
		if err != nil {
			log.Error(err, "Failed to create new API Service", "Service.Namespace", newApiSvc.Namespace, "Service.Name", newApiSvc.Name)
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorCreatingApiService", "Failed to create "+pulp.Name+"-api-svc service: "+err.Error())
			return ctrl.Result{}, err
		}
		// Service created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get API Service")
		return ctrl.Result{}, err
	}

	// Ensure the service spec is as expected
	if !equality.Semantic.DeepDerivative(newApiSvc.Spec, apiSvc.Spec) {
		log.Info("The API service has been modified! Reconciling ...")
		r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "UpdatingApiService", "Reconciling "+pulp.Name+"-api-svc service")
		err = r.Update(ctx, newApiSvc)
		if err != nil {
			log.Error(err, "Error trying to update the API Service object ... ")
			r.updateStatus(ctx, pulp, metav1.ConditionFalse, pulp.Spec.DeploymentType+"-API-Ready", "ErrorUpdatingApiService", "Failed to reconcile "+pulp.Name+"-api-svc service: "+err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// pulpController, err := r.checkRunningPods(ctx, pulp, labelsForPulpApi(pulp), log)
	// if err != nil {
	// 	return pulpController, err
	// } else if pulpController.Requeue {
	// 	return pulpController, nil
	// } else if pulpController.RequeueAfter > 0 {
	// 	return pulpController, nil
	// }

	//r.updateStatus(ctx, pulp, metav1.ConditionTrue, pulp.Spec.DeploymentType+"-API-Ready", "ApiTasksFinished", "All API tasks ran successfully")
	return ctrl.Result{}, nil
}

// fileStoragePVC returns a PVC object
func (r *PulpReconciler) fileStoragePVC(m *repomanagerv1alpha1.Pulp) *corev1.PersistentVolumeClaim {

	// Define the new PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-file-storage",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       m.Spec.DeploymentType + "-storage",
				"app.kubernetes.io/instance":   m.Spec.DeploymentType + "-storage-" + m.Name,
				"app.kubernetes.io/component":  "storage",
				"app.kubernetes.io/part-of":    m.Spec.DeploymentType,
				"app.kubernetes.io/managed-by": m.Spec.DeploymentType + "-operator",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resource.MustParse(m.Spec.FileStorageSize),
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode(m.Spec.FileStorageAccessMode),
			},
		},
	}

	if len(m.Spec.FileStorageClass) > 0 {
		pvc.Spec.StorageClassName = &m.Spec.FileStorageClass
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, pvc, r.Scheme)

	return pvc

}

// deploymentForPulpApi returns a pulp-api Deployment object
func (r *PulpReconciler) deploymentForPulpApi(m *repomanagerv1alpha1.Pulp) *appsv1.Deployment {

	replicas := m.Spec.Api.Replicas
	ls := labelsForPulpApi(m)

	affinity := &corev1.Affinity{}
	if m.Spec.Api.Affinity.NodeAffinity != nil {
		affinity.NodeAffinity = m.Spec.Api.Affinity.NodeAffinity
	}

	runAsUser := int64(0)
	fsGroup := int64(0)
	podSecurityContext := &corev1.PodSecurityContext{}
	IsOpenShift, _ := controllers.IsOpenShift()
	if !IsOpenShift {
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

	var dbHost, dbPort string
	if reflect.DeepEqual(m.Spec.Database.ExternalDB, repomanagerv1alpha1.ExternalDB{}) {
		containerPort := 0
		if m.Spec.Database.PostgresPort == 0 {
			containerPort = 5432
		} else {
			containerPort = m.Spec.Database.PostgresPort
		}
		dbHost = m.Name + "-database-svc"
		dbPort = strconv.Itoa(containerPort)
	} else {
		dbHost = m.Spec.Database.ExternalDB.PostgresHost
		dbPort = strconv.Itoa(m.Spec.Database.ExternalDB.PostgresPort)
	}

	envVars := []corev1.EnvVar{
		{Name: "POSTGRES_SERVICE_HOST", Value: dbHost},
		{Name: "POSTGRES_SERVICE_PORT", Value: dbPort},
		{Name: "PULP_GUNICORN_TIMEOUT", Value: strconv.Itoa(m.Spec.Api.GunicornTimeout)},
		{Name: "PULP_API_WORKERS", Value: strconv.Itoa(m.Spec.Api.GunicornWorkers)},
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

	dbFieldsEncryptionSecret := ""
	if m.Spec.DBFieldsEncryptionSecret == "" {
		dbFieldsEncryptionSecret = m.Name + "-db-fields-encryption"
	} else {
		dbFieldsEncryptionSecret = m.Spec.DBFieldsEncryptionSecret
	}

	adminSecretName := m.Name + "-admin-password"
	if len(m.Spec.AdminPasswordSecret) > 1 {
		adminSecretName = m.Spec.AdminPasswordSecret
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

	// issue #526
	// if .Spec.FileStorageClass == "" and there is no default SC defined
	// we should mount an emptyDir volume in pod
	if len(m.Spec.ObjectStorageAzureSecret) == 0 && len(m.Spec.ObjectStorageS3Secret) == 0 {
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
			{
				Name: "assets-file-storage",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
		volumes = append(volumes, emptyDir...)
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

	var containerAuthSecretName string
	if m.Spec.ContainerTokenSecret != "" {
		containerAuthSecretName = m.Spec.ContainerTokenSecret
	} else {
		containerAuthSecretName = m.Name + "-container-auth"
	}

	containerTokenSecretVolume := corev1.Volume{
		Name: m.Name + "-container-auth-certs",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: containerAuthSecretName,
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

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      m.Name + "-server",
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
			Name:      m.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
	}

	if len(m.Spec.ObjectStorageAzureSecret) == 0 && len(m.Spec.ObjectStorageS3Secret) == 0 {
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
			{
				Name:      "assets-file-storage",
				MountPath: "/var/lib/pulp/assets",
			},
		}
		volumeMounts = append(volumeMounts, emptyDir...)
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

	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/usr/bin/readyz.py",
					getPulpSetting(m, "api_root") + "api/v3/status/",
				},
			},
		},
		FailureThreshold:    10,
		InitialDelaySeconds: 60,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
	}

	livenessProbe := &corev1.Probe{
		FailureThreshold: 5,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: getPulpSetting(m, "api_root") + "api/v3/status/",
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

	// the following variables are defined to avoid issues with reconciliation
	restartPolicy := corev1.RestartPolicy("Always")
	terminationGracePeriodSeconds := int64(30)
	dnsPolicy := corev1.DNSPolicy("ClusterFirst")
	schedulerName := corev1.DefaultSchedulerName

	// deployment definition
	dep := &appsv1.Deployment{
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
				"app":                          "pulp-api",
				"pulp_cr":                      m.Name,
				"owner":                        "pulp-dev",
			},
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
					ServiceAccountName:        "pulp-operator-controller-manager",
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

						LivenessProbe:  livenessProbe,
						ReadinessProbe: readinessProbe,
						Resources:      resources,
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
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
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

// pulpServerSecret creates the pulp-server secret object which is used to
// populate the /etc/pulp/settings.py config file
func (r *PulpReconciler) pulpServerSecret(ctx context.Context, m *repomanagerv1alpha1.Pulp, log logr.Logger) *corev1.Secret {

	var dbHost, dbPort, dbUser, dbPass, dbName, dbSSLMode string

	// if there is no external database configuration get the databaseconfig from pulp-postgres-configuration secret
	if reflect.DeepEqual(m.Spec.Database.ExternalDB, repomanagerv1alpha1.ExternalDB{}) {
		log.Info("Retrieving Postgres credentials from "+m.Name+"-postgres-configuration secret", "Secret.Namespace", m.Namespace, "Secret.Name", m.Name)
		pgCredentials, err := r.retrieveSecretData(ctx, m.Name+"-postgres-configuration", m.Namespace, true, "username", "password", "database", "port", "sslmode")
		if err != nil {
			log.Error(err, "Secret Not Found!", "Secret.Namespace", m.Namespace, "Secret.Name", m.Name)
		}
		dbHost = m.Name + "-database-svc"
		dbPort = pgCredentials["port"]
		dbUser = pgCredentials["username"]
		dbPass = pgCredentials["password"]
		dbName = pgCredentials["database"]
		dbSSLMode = pgCredentials["sslmode"]
	} else {
		dbHost = m.Spec.Database.ExternalDB.PostgresHost
		dbPort = strconv.Itoa(m.Spec.Database.ExternalDB.PostgresPort)
		dbUser = m.Spec.Database.ExternalDB.PostgresUser
		dbPass = m.Spec.Database.ExternalDB.PostgresPassword
		dbName = m.Spec.Database.ExternalDB.PostgresDBName
		dbSSLMode = m.Spec.Database.ExternalDB.PostgresSSLMode
	}

	// default settings.py configuration
	var pulp_settings = `CACHE_ENABLED = "True"
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
ANSIBLE_API_HOSTNAME = "http://` + m.Name + `-web-svc.` + m.Namespace + `.svc.cluster.local:24880"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys/"
CONTENT_ORIGIN = "http://` + m.Name + `-web-svc.` + m.Namespace + `.svc.cluster.local:24880"
DATABASES = {
	'default': {
		'HOST': '` + dbHost + `',
		'ENGINE': 'django.db.backends.postgresql_psycopg2',
		'NAME': '` + dbName + `',
		'USER': '` + dbUser + `',
		'PASSWORD': '` + dbPass + `',
		'PORT': '` + dbPort + `',
		'CONN_MAX_AGE': 0,
		'OPTIONS': { 'sslmode': '` + dbSSLMode + `' },
	}
}
GALAXY_FEATURE_FLAGS = {
	'execution_environments': 'True',
}
PRIVATE_KEY_PATH = "/etc/pulp/keys/container_auth_private_key.pem"
PUBLIC_KEY_PATH = "/etc/pulp/keys/container_auth_public_key.pem"
REDIS_HOST =  "` + m.Name + `-redis-svc.` + m.Namespace + `"
REDIS_PORT =  "6379"
REDIS_PASSWORD = ""
TOKEN_AUTH_DISABLED = "False"
TOKEN_SERVER = "http://` + m.Name + `-web-svc.` + m.Namespace + `.svc.cluster.local:24880/token/"
TOKEN_SIGNATURE_ALGORITHM = "ES256"
`

	pulp_settings = pulp_settings + fmt.Sprintln("API_ROOT = \"/pulp/\"")

	if len(m.Spec.ObjectStorageAzureSecret) > 0 {
		log.Info("Retrieving Azure data from " + m.Spec.ObjectStorageAzureSecret)
		storageData, err := r.retrieveSecretData(ctx, m.Spec.ObjectStorageAzureSecret, m.Namespace, true, "azure-account-name", "azure-account-key", "azure-container", "azure-container-path", "azure-connection-string")
		if err != nil {
			log.Error(err, "Secret Not Found!", "Secret.Namespace", m.Namespace, "Secret.Name", m.Spec.ObjectStorageAzureSecret)
		}
		pulp_settings = pulp_settings + `AZURE_CONNECTION_STRING = '` + storageData["azure-connection-string"] + `'
AZURE_LOCATION = '` + storageData["azure-container-path"] + `'
AZURE_ACCOUNT_NAME = '` + storageData["azure-account-name"] + `'
AZURE_ACCOUNT_KEY = '` + storageData["azure-account-key"] + `'
AZURE_CONTAINER = '` + storageData["azure-container"] + `'
AZURE_URL_EXPIRATION_SECS = 60
AZURE_OVERWRITE_FILES = "True"
DEFAULT_FILE_STORAGE = "storages.backends.azure_storage.AzureStorage"
`
	}

	if len(m.Spec.ObjectStorageS3Secret) > 0 {
		log.Info("Retrieving S3 data from " + m.Spec.ObjectStorageS3Secret)
		storageData, err := r.retrieveSecretData(ctx, m.Spec.ObjectStorageS3Secret, m.Namespace, true, "s3-access-key-id", "s3-secret-access-key", "s3-bucket-name", "s3-region")
		if err != nil {
			log.Error(err, "Secret Not Found!", "Secret.Namespace", m.Namespace, "Secret.Name", m.Spec.ObjectStorageS3Secret)
		}

		optionalKey, err := r.retrieveSecretData(ctx, m.Spec.ObjectStorageS3Secret, m.Namespace, false, "s3-endpoint")
		if err == nil {
			pulp_settings = pulp_settings + fmt.Sprintf("AWS_S3_ENDPOINT_URL = \"%v\"\n", optionalKey["s3-endpoint"])
		}

		pulp_settings = pulp_settings + `AWS_ACCESS_KEY_ID = '` + storageData["s3-access-key-id"] + `'
AWS_SECRET_ACCESS_KEY = '` + storageData["s3-secret-access-key"] + `'
AWS_STORAGE_BUCKET_NAME = '` + storageData["s3-bucket-name"] + `'
AWS_S3_REGION_NAME = '` + storageData["s3-region"] + `'
AWS_DEFAULT_ACL = "@none None"
S3_USE_SIGV4 = "True"
AWS_S3_SIGNATURE_VERSION = "s3v4"
AWS_S3_ADDRESSING_STYLE = "path"
DEFAULT_FILE_STORAGE = "storages.backends.s3boto3.S3Boto3Storage"
MEDIA_ROOT = ""
`
	}

	// configure settings.py with keycloak integration variables
	if len(m.Spec.SSOSecret) > 0 {
		r.ssoConfig(ctx, m, &pulp_settings)
	}

	pulp_settings = addCustomPulpSettings(m, pulp_settings)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-server",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"settings.py": pulp_settings,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, sec, r.Scheme)
	return sec
}

// pulp-db-fields-encryption secret
func pulpDBFieldsEncryptionSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-db-fields-encryption",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"database_fields.symmetric.key": "81HqDtbqAywKSOumSha3BhWNOdQ26slT6K0YaZeZyPs=",
		},
	}

}

// pulp-admin-passowrd
func pulpAdminPasswordSecret(m *repomanagerv1alpha1.Pulp) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-admin-password",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"password": createPwd(32),
		},
	}
}

// pulp-container-auth
func pulpContainerAuth(m *repomanagerv1alpha1.Pulp) *corev1.Secret {

	privKey, pubKey := genTokenAuthKey()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name + "-container-auth",
			Namespace: m.Namespace,
		},
		StringData: map[string]string{
			"container_auth_private_key.pem": privKey,
			"container_auth_public_key.pem":  pubKey,
		},
	}
}

// serviceForAPI returns a service object for pulp-api
func (r *PulpReconciler) serviceForAPI(m *repomanagerv1alpha1.Pulp) *corev1.Service {

	svc := serviceAPIObject(m.Name, m.Namespace, m.Spec.DeploymentType)

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func serviceAPIObject(name, namespace, deployment_type string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-api-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       deployment_type + "-api",
				"app.kubernetes.io/instance":   deployment_type + "-api-" + name,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    deployment_type,
				"app.kubernetes.io/managed-by": deployment_type + "-operator",
				"app":                          "pulp-api",
				"pulp_cr":                      name,
			},
		},
		Spec: serviceAPISpec(name, namespace, deployment_type),
	}
}

// api service spec
func serviceAPISpec(name, namespace, deployment_type string) corev1.ServiceSpec {

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
			Name:       "api-24817",
			Port:       24817,
			Protocol:   servicePortProto,
			TargetPort: targetPort,
		}},
		Selector: map[string]string{
			"app.kubernetes.io/name":       deployment_type + "-api",
			"app.kubernetes.io/instance":   deployment_type + "-api-" + name,
			"app.kubernetes.io/component":  "api",
			"app.kubernetes.io/part-of":    deployment_type,
			"app.kubernetes.io/managed-by": deployment_type + "-operator",
			"app":                          "pulp-api",
			"pulp_cr":                      name,
		},
		SessionAffinity: serviceAffinity,
		Type:            serviceType,
	}
}
