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

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	jobs "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	adminPasswordSecretName = "admin-password"
)

// pulpApiController provision and reconciles api objects
func (r *RepoManagerReconciler) updateAdminSecret(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	log := r.RawLogger

	adminSecretName := controllers.GetAdminSecretName(*pulp)
	adminSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: pulp.Namespace}, adminSecret); err != nil {
		log.Error(err, "Failed to find "+adminPasswordSecretName+" Secret!")
	}

	// if the secret didn't change there is nothing to do
	calculatedHash := controllers.CalculateHash(adminSecret.Data)
	currentHash := controllers.GetCurrentHash(adminSecret)
	if currentHash == calculatedHash {
		return
	}

	containers := []corev1.Container{resetAdminPasswordContainer(pulp)}
	volumes := resetAdminPasswordVolumes(pulp, adminSecretName)
	backOffLimit := int32(2)
	jobTTL := int32(3600)

	// job definition
	job := &jobs.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "reset-admin-password-",
			Namespace:    pulp.Namespace,
		},
		Spec: jobs.JobSpec{
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &jobTTL,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: "Never",
					Containers:    containers,
					Volumes:       volumes,
				},
			},
		},
	}

	// create job
	log.Info("Creating a new " + adminPasswordSecretName + " reset Job ...")
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create "+adminPasswordSecretName+" Job!")
	}

	// update secret hash label
	log.V(1).Info("Updating " + adminPasswordSecretName + " hash label ...")
	controllers.SetHashLabel(calculatedHash, adminSecret)
	if err := r.Update(ctx, adminSecret); err != nil {
		log.Error(err, "Failed to update "+adminPasswordSecretName+" Secret label!")
	}
}

// resetAdminPasswordVolumes defines the list of volumeMounts from reset admin password container
func resetAdminPasswordVolumes(pulp *repomanagerpulpprojectorgv1beta2.Pulp, adminSecretName string) []corev1.Volume {
	dbFieldsEncryptionSecret := controllers.GetDBFieldsEncryptionSecret(*pulp)

	return []corev1.Volume{
		{
			Name: pulp.Name + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: pulp.Name + "-server",
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
			Name: pulp.Name + "-db-fields-encryption",
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
}

// resetAdminPasswordVolumeMounts defines the list of volumeMounts from reset admin password container
func resetAdminPasswordVolumeMounts(pulp *repomanagerpulpprojectorgv1beta2.Pulp) []corev1.VolumeMount {
	// admin password secret volume
	adminSecretName := controllers.GetAdminSecretName(*pulp)

	return []corev1.VolumeMount{
		{
			Name:      pulp.Name + "-server",
			MountPath: "/etc/pulp/settings.py",
			SubPath:   "settings.py",
			ReadOnly:  true,
		},
		{
			Name:      pulp.Name + "-db-fields-encryption",
			MountPath: "/etc/pulp/keys/database_fields.symmetric.key",
			SubPath:   "database_fields.symmetric.key",
			ReadOnly:  true,
		},
		{
			Name:      adminSecretName,
			MountPath: "/etc/pulp/pulp-admin-password",
			SubPath:   "admin-password",
			ReadOnly:  true,
		},
	}
}

// resetAdminPasswordResources returns the resourceRequirements for reset admin password container
func resetAdminPasswordResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("20m"),
			corev1.ResourceMemory: resource.MustParse("56Mi"),
		},
	}
}

// resetAdminPasswordContainer defines the container spec for the reset admin password job
func resetAdminPasswordContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp) corev1.Container {
	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)

	// volume mounts
	volumeMounts := resetAdminPasswordVolumeMounts(pulp)

	// resource requirements
	resources := resetAdminPasswordResources()

	return corev1.Container{
		Name:    "reset-admin-password",
		Image:   pulp.Spec.Image + ":" + pulp.Spec.ImageVersion,
		Env:     envVars,
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			`/usr/bin/wait_on_postgres.py
			/usr/bin/wait_on_database_migrations.sh
			ADMIN_PASSWORD_FILE=/etc/pulp/pulp-admin-password
			if [[ -f "$ADMIN_PASSWORD_FILE" ]]; then
			  echo "pulp admin can be initialized."
			  PULP_ADMIN_PASSWORD=$(cat $ADMIN_PASSWORD_FILE)
			fi
			if [ -n "${PULP_ADMIN_PASSWORD}" ]; then
			  /usr/local/bin/pulpcore-manager reset-admin-password --password "${PULP_ADMIN_PASSWORD}"
			fi`,
		},
		Resources:    resources,
		VolumeMounts: volumeMounts,
	}
}
