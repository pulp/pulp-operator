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
	"encoding/json"
	"reflect"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// updateAdminPasswordJob creates a k8s job if the admin-password secret has changed
func (r *RepoManagerReconciler) updateAdminPasswordJob(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	log := r.RawLogger

	adminSecretName := controllers.GetAdminSecretName(*pulp)
	adminSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: pulp.Namespace}, adminSecret); err != nil {
		log.Error(err, "Failed to find "+adminSecretName+" Secret!")
	}

	// if the secret didn't change there is nothing to do
	calculatedHash := controllers.CalculateHash(adminSecret.Data)
	currentHash := controllers.GetCurrentHash(adminSecret)
	if currentHash == calculatedHash {
		return
	}

	jobName := settings.ResetAdminPwdJob(pulp.Name)
	labels := jobLabels(*pulp)
	labels["app.kubernetes.io/component"] = "reset-admin-password"
	containers := []corev1.Container{resetAdminPasswordContainer(pulp)}
	volumes := pulpcoreVolumes(pulp, adminSecretName)
	backOffLimit := int32(2)
	jobTTL := int32(3600)

	// job definition
	job := commonJob(pulpJobConfig{
		jobName,
		pulp.Namespace,
		settings.PulpServiceAccount(pulp.Name),
		labels,
		&backOffLimit,
		&jobTTL,
		containers,
		volumes,
	})

	ctrl.SetControllerReference(pulp, job, r.Scheme)
	// create job
	log.Info("Creating " + jobName + "* Job")
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create "+jobName+"* Job!")
	}

	// update secret hash label
	log.V(1).Info("Updating " + adminSecretName + " hash label ...")
	controllers.SetHashLabel(calculatedHash, adminSecret)
	if err := r.Update(ctx, adminSecret); err != nil {
		log.Error(err, "Failed to update "+adminSecretName+" Secret label!")
	}
}

// pulpcoreVolumes defines the list of volumes used by pulpcore containers
func pulpcoreVolumes(pulp *repomanagerpulpprojectorgv1beta2.Pulp, adminSecretName string) []corev1.Volume {
	dbFieldsEncryptionSecret := controllers.GetDBFieldsEncryptionSecret(*pulp)

	volumes := []corev1.Volume{
		{
			Name: pulp.Name + "-server",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: settings.PulpServerSecret(pulp.Name),
					Items: []corev1.KeyToPath{{
						Key:  "settings.py",
						Path: "settings.py",
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

	if len(adminSecretName) > 0 {
		adminSecret := corev1.Volume{
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
		}
		return append(volumes, adminSecret)
	}

	return volumes
}

// pulpcoreVolumeMounts defines the list of volumeMounts from pulpcore containers
func pulpcoreVolumeMounts(pulp *repomanagerpulpprojectorgv1beta2.Pulp) []corev1.VolumeMount {
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
	}
}

// resetAdminPasswordContainer defines the container spec for the reset admin password job
func resetAdminPasswordContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp) corev1.Container {
	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)
	envVars = append(envVars, controllers.SetCustomEnvVars(*pulp, "AdminPasswordJob")...)

	// volume mounts
	volumeMounts := pulpcoreVolumeMounts(pulp)

	// admin password secret volume
	adminSecretName := controllers.GetAdminSecretName(*pulp)
	adminSecretVolume := corev1.VolumeMount{
		Name:      adminSecretName,
		MountPath: "/etc/pulp/pulp-admin-password",
		SubPath:   "admin-password",
		ReadOnly:  true,
	}
	volumeMounts = append(volumeMounts, adminSecretVolume)

	// resource requirements
	resources := pulp.Spec.AdminPasswordJob.PulpContainer.ResourceRequirements

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
		Resources:       resources,
		VolumeMounts:    volumeMounts,
		SecurityContext: controllers.SetDefaultSecurityContext(),
	}
}

// migrationJob creates a k8s Job to run django migrations
func (r *RepoManagerReconciler) migrationJob(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	log := r.RawLogger

	labels := jobLabels(*pulp)
	labels["app.kubernetes.io/component"] = "migration"
	containers := []corev1.Container{migrationContainer(pulp)}
	volumes := pulpcoreVolumes(pulp, "")
	backOffLimit := int32(2)
	jobTTL := int32(3600)

	// job definition
	job := commonJob(pulpJobConfig{
		settings.MigrationJob(pulp.Name),
		pulp.Namespace,
		settings.PulpServiceAccount(pulp.Name),
		labels,
		&backOffLimit,
		&jobTTL,
		containers,
		volumes,
	})

	ctrl.SetControllerReference(pulp, job, r.Scheme)
	// create the Job
	log.Info("Creating a new pulpcore migration Job")
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create pulpcore migration Job!")
	}
}

// migrationContainer defines the container spec for the django migrations Job
func migrationContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp) corev1.Container {
	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)
	envVars = append(envVars, controllers.SetCustomEnvVars(*pulp, "MigrationJob")...)

	// volume mounts
	volumeMounts := pulpcoreVolumeMounts(pulp)

	// resource requirements
	resources := pulp.Spec.MigrationJob.PulpContainer.ResourceRequirements

	return corev1.Container{
		Name:            "migration",
		Image:           pulp.Spec.Image + ":" + pulp.Spec.ImageVersion,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Env:             envVars,
		Command:         []string{"/bin/sh"},
		Args: []string{
			"-c",
			`/usr/bin/wait_on_postgres.py
/usr/local/bin/pulpcore-manager migrate --noinput`,
		},
		Resources:       resources,
		VolumeMounts:    volumeMounts,
		SecurityContext: controllers.SetDefaultSecurityContext(),
	}
}

// jobLabels defines the common labels used in Jobs
func jobLabels(pulp repomanagerpulpprojectorgv1beta2.Pulp) map[string]string {
	return settings.CommonLabels(pulp)
}

// updateContentChecksumsJob creates a k8s Job to update the list of allowed content checksums
func (r *RepoManagerReconciler) updateContentChecksumsJob(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	log := r.RawLogger

	if !contentChecksumsModified(pulp) {
		return
	}

	jobName := settings.UpdateChecksumsJob(pulp.Name)
	labels := jobLabels(*pulp)
	labels["app.kubernetes.io/component"] = "allowed-content-checksums"
	containers := []corev1.Container{contentChecksumsContainer(pulp)}
	volumes := pulpcoreVolumes(pulp, "")
	backOffLimit := int32(2)
	jobTTL := int32(60)

	// job definition
	job := commonJob(pulpJobConfig{
		jobName,
		pulp.Namespace,
		settings.PulpServiceAccount(pulp.Name),
		labels,
		&backOffLimit,
		&jobTTL,
		containers,
		volumes,
	})

	ctrl.SetControllerReference(pulp, job, r.Scheme)
	// create the Job
	log.Info("Creating a new " + jobName + "* Job")
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create "+jobName+"* Job!")
	}

	// update .status
	settings, _ := json.Marshal(pulp.Spec.AllowedContentChecksums)
	pulp.Status.AllowedContentChecksums = string(settings)
	r.Status().Update(ctx, pulp)
}

// contentChecksumsContainer defines the container spec for the updateContentChecksums Job
func contentChecksumsContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp) corev1.Container {
	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)
	envVars = append(envVars, controllers.SetCustomEnvVars(*pulp, string(settings.API))...)

	// volume mounts
	volumeMounts := pulpcoreVolumeMounts(pulp)

	// resource requirements
	resources := pulp.Spec.MigrationJob.PulpContainer.ResourceRequirements

	return corev1.Container{
		Name:            "update-checksum",
		Image:           pulp.Spec.Image + ":" + pulp.Spec.ImageVersion,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Env:             envVars,
		Command:         []string{"/bin/sh"},
		Args: []string{
			"-c",
			`/usr/bin/wait_on_postgres.py
			/usr/bin/wait_on_database_migrations.sh
			pulpcore-manager handle-artifact-checksums`,
		},
		Resources:       resources,
		VolumeMounts:    volumeMounts,
		SecurityContext: controllers.SetDefaultSecurityContext(),
	}
}

// contentChecksumsModified returns true if
// .status.AllowedContentChecksums != pulp.Spec.AllowedContentChecksums
func contentChecksumsModified(pulp *repomanagerpulpprojectorgv1beta2.Pulp) bool {
	var statusAllowedChecksum []string
	json.Unmarshal([]byte(pulp.Status.AllowedContentChecksums), &statusAllowedChecksum)
	return !reflect.DeepEqual(pulp.Spec.AllowedContentChecksums, statusAllowedChecksum)
}

// signingScriptContainer returns the definition of the container used in signingScript Job
func signingScriptContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp, scriptsSecret corev1.Secret, r RepoManagerReconciler) corev1.Container {
	// volume mounts
	volumeMounts := pulpcoreVolumeMounts(pulp)
	signingSecretMount := []corev1.VolumeMount{
		{
			Name:      pulp.Name + "-signing-scripts",
			MountPath: "/var/lib/pulp/scripts/" + settings.CollectionSigningScriptName,
			SubPath:   settings.CollectionSigningScriptName,
			ReadOnly:  true,
		},
		{
			Name:      pulp.Name + "-signing-scripts",
			MountPath: "/var/lib/pulp/scripts/" + settings.ContainerSigningScriptName,
			SubPath:   settings.ContainerSigningScriptName,
			ReadOnly:  true,
		},
		{
			Name:      "gpg-keys",
			MountPath: "/etc/pulp/keys/signing_service.gpg",
			SubPath:   "signing_service.gpg",
			ReadOnly:  true,
		},
		{
			Name:      "ephemeral-gpg",
			MountPath: "/var/lib/pulp/.gnupg",
		},
	}
	volumeMounts = append(volumeMounts, signingSecretMount...)

	// resource requirements
	resources := pulp.Spec.SigningJob.PulpContainer.ResourceRequirements

	fingerprint, _ := controllers.GetSigningKeyFingerprint(r.Client, pulp.Spec.SigningSecret, pulp.Namespace)

	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)
	envVars = append(envVars, controllers.SetCustomEnvVars(*pulp, "SigningJob")...)
	envVars = append(envVars, corev1.EnvVar{Name: "PULP_SIGNING_KEY_FINGERPRINT", Value: fingerprint})
	envVars = append(envVars, corev1.EnvVar{Name: "HOME", Value: "/var/lib/pulp"})

	args := []string{
		`gpg --batch --import /etc/pulp/keys/signing_service.gpg
echo "${PULP_SIGNING_KEY_FINGERPRINT}:6" | gpg --import-ownertrust
`,
	}
	if controllers.DeployCollectionSign(scriptsSecret) {
		args[0] += "/usr/local/bin/pulpcore-manager remove-signing-service collection-signing-service\n"
		args[0] += "/usr/local/bin/pulpcore-manager add-signing-service collection-signing-service /var/lib/pulp/scripts/" + settings.CollectionSigningScriptName + " " + fingerprint + "\n"
		envVars = append(envVars, corev1.EnvVar{Name: "COLLECTION_SIGNING_SERVICE", Value: "collection-signing-service"})
	}
	if controllers.DeployContainerSign(scriptsSecret) {
		args[0] += "/usr/local/bin/pulpcore-manager remove-signing-service container-signing-service --class container:ManifestSigningService\n"
		args[0] += "/usr/local/bin/pulpcore-manager add-signing-service container-signing-service /var/lib/pulp/scripts/" + settings.ContainerSigningScriptName + " " + fingerprint + " --class container:ManifestSigningService"
		envVars = append(envVars, corev1.EnvVar{Name: "CONTAINER_SIGNING_SERVICE", Value: "container-signing-service"})
	}

	return corev1.Container{
		Name:            "signing-metadata",
		Image:           signingScriptContainerImage(*pulp),
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Env:             envVars,
		Command:         []string{"/bin/sh", "-c"},
		Args:            args,
		Resources:       resources,
		VolumeMounts:    volumeMounts,
		SecurityContext: controllers.SetDefaultSecurityContext(),
	}
}

// signingScriptJobVolumes returns the list of volumes used by signingScript Job pod
func signingScriptJobVolumes(pulp *repomanagerpulpprojectorgv1beta2.Pulp, secret corev1.Secret) []corev1.Volume {
	secretItems := []corev1.KeyToPath{}
	if controllers.DeployCollectionSign(secret) {
		item := corev1.KeyToPath{Key: settings.CollectionSigningScriptName, Path: settings.CollectionSigningScriptName}
		secretItems = append(secretItems, item)
	}
	if controllers.DeployContainerSign(secret) {
		item := corev1.KeyToPath{Key: settings.ContainerSigningScriptName, Path: settings.ContainerSigningScriptName}
		secretItems = append(secretItems, item)
	}

	volumes := pulpcoreVolumes(pulp, "")
	volumePermissions := int32(0755)
	signingSecretVolume := corev1.Volume{
		Name: pulp.Name + "-signing-scripts",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  pulp.Spec.SigningScripts,
				Items:       secretItems,
				DefaultMode: &volumePermissions,
			},
		},
	}

	gpgKeysVolume := corev1.Volume{
		Name: "gpg-keys",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: pulp.Spec.SigningSecret,
				Items:      []corev1.KeyToPath{{Key: "signing_service.gpg", Path: "signing_service.gpg"}},
			},
		},
	}

	tmp := corev1.Volume{
		Name: "ephemeral-gpg",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	return append(volumes, signingSecretVolume, gpgKeysVolume, tmp)
}

// signingScriptJob creates a k8s job to store the signing scripts
func (r *RepoManagerReconciler) signingScriptJob(ctx context.Context, pulp *repomanagerpulpprojectorgv1beta2.Pulp) {
	log := r.RawLogger

	secretName := pulp.Spec.SigningScripts
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: pulp.Namespace}, secret); err != nil {
		log.Error(err, "Failed to find "+secretName+" Secret!")
	}

	jobName := settings.SigningScriptJob(pulp.Name)
	labels := jobLabels(*pulp)
	labels["app.kubernetes.io/component"] = "signing-script"
	backoffLimit := int32(2)
	jobTTL := int32(360)

	job := commonJob(pulpJobConfig{
		jobName,
		pulp.Namespace,
		settings.PulpServiceAccount(pulp.Name),
		labels,
		&backoffLimit,
		&jobTTL,
		[]corev1.Container{signingScriptContainer(pulp, *secret, *r)},
		signingScriptJobVolumes(pulp, *secret),
	})

	job.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer(pulp, pulp.Spec.SigningJob.PulpContainer.ResourceRequirements, signingScriptContainerImage(*pulp))}

	ctrl.SetControllerReference(pulp, job, r.Scheme)
	log.Info("Creating a new " + jobName + "* Job")
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create "+jobName+"* Job!")
	}

	log.V(1).Info("Updating " + secretName + " hash label ...")
	calculatedHash := controllers.CalculateHash(secret.Data)
	controllers.SetHashLabel(calculatedHash, secret)
	if err := r.Update(ctx, secret); err != nil {
		log.Error(err, "Failed to update "+secretName+" Secret label!")
	}
}

// signingScriptContainerImage defines the container image to use in signingScript Job containers
func signingScriptContainerImage(pulp repomanagerpulpprojectorgv1beta2.Pulp) string {
	image := pulp.Spec.SigningJob.PulpContainer.Image
	if len(image) == 0 {
		image = pulp.Spec.Image + ":" + pulp.Spec.ImageVersion
	}
	return image
}

// initContainer returns the definition of a common container with instructions to check if the database migrations finished
func initContainer(pulp *repomanagerpulpprojectorgv1beta2.Pulp, resources corev1.ResourceRequirements, image string) corev1.Container {
	args := []string{
		"-c",
		`/usr/bin/wait_on_postgres.py
/usr/bin/wait_on_database_migrations.sh`,
	}

	// env vars
	envVars := controllers.GetPostgresEnvVars(*pulp)
	envVars = append(envVars, controllers.SetCustomEnvVars(*pulp, "SigningJob")...)

	volumeMounts := []corev1.VolumeMount{
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
	}

	allowPrivilegeEscalation, runAsNonRoot := false, true
	securityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: &runAsNonRoot,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	return corev1.Container{
		Name:            "init-container",
		Image:           image,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Env:             envVars,
		Command:         []string{"/bin/sh"},
		Args:            args,
		VolumeMounts:    volumeMounts,
		Resources:       resources,
		SecurityContext: securityContext,
	}
}

type pulpJobConfig struct {
	name                    string
	namespace               string
	saName                  string
	labels                  map[string]string
	backoffLimit            *int32
	ttlSecondsAfterFinished *int32
	containers              []corev1.Container
	volumes                 []corev1.Volume
}

// commonJob returns a k8s Job with a common resource definition
func commonJob(jobConfig pulpJobConfig) *batchv1.Job {

	securityContext := &corev1.PodSecurityContext{}
	if isOpenShift, _ := controllers.IsOpenShift(); !isOpenShift {
		runAsUser := int64(700)
		fsGroup := int64(700)
		securityContext = &corev1.PodSecurityContext{RunAsUser: &runAsUser, FSGroup: &fsGroup}
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobConfig.name,
			Namespace:    jobConfig.namespace,
			Labels:       jobConfig.labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            jobConfig.backoffLimit,
			TTLSecondsAfterFinished: jobConfig.ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: jobConfig.labels},
				Spec: corev1.PodSpec{
					RestartPolicy:      "Never",
					Containers:         jobConfig.containers,
					Volumes:            jobConfig.volumes,
					ServiceAccountName: jobConfig.saName,
					SecurityContext:    securityContext,
				},
			},
		},
	}
}
