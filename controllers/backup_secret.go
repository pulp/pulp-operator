package controllers

import (
	"context"

	"gopkg.in/yaml.v3"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type adminSecret struct {
	Admin_password_name string `json:"admin_password_name"`
	Admin_password      string `json:"admin_password"`
	Database_password   string `json:"database_password"`
	Database_username   string `json:"database_username"`
	Database_name       string `json:"database_name"`
	Database_port       string `json:"Database_port"`
	Database_host       string `json:"database_host"`
	Database_type       string `json:"database_type"`
	Database_sslmode    string `json:"database_sslmode"`
	Postgres_version    string `json:"postgres_version"`
	Db_secret_name      string `json:"db_secret_name,omitempty"`
}

type encryptionFieldSecret struct {
	Db_fields_encryption_secret string `json:"db_fields_encryption_secret"`
	Db_fields_encryption_key    string `json:"db_fields_encryption_key"`
}

type signingSecret struct {
	Signing_secret      string `json:"signing_secret"`
	Signing_service_gpg string `json:"signing_service_gpg"`
	Signing_service_asc string `json:"signing_service_asc"`
}

type containerTokenSecret struct {
	Container_token_secret     string `json:"container_token_secret"`
	Container_auth_private_key string `json:"container_auth_private_key"`
	Container_auth_public_key  string `json:"container_auth_public_key"`
}

func (r *PulpBackupReconciler) backupSecret(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) {
	log := ctrllog.FromContext(ctx)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName, Namespace: pulpBackup.Namespace}, pulp)

	// ADMIN AND POSTGRES SECRETS
	secretData := adminSecret{}
	adminPwd := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.AdminPasswordSecret, Namespace: pulpBackup.Namespace}, adminPwd)
	if err != nil {
		log.Error(err, "Error trying to find pulp-admin-password secret")
		//return ctrl.Result{}, err
	}

	secretData.Admin_password_name = pulpBackup.Spec.AdminPasswordSecret
	secretData.Admin_password = string(adminPwd.Data["password"])

	pgSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-postgres-configuration", Namespace: pulpBackup.Namespace}, pgSecret)
	if err != nil {
		log.Error(err, "Error trying to find pulp-postgres-configuration secret")
		//return ctrl.Result{}, err
	}

	secretData.Database_password = string(pgSecret.Data["password"])
	secretData.Database_username = string(pgSecret.Data["username"])
	secretData.Database_name = string(pgSecret.Data["database"])
	secretData.Database_port = string(pgSecret.Data["port"])
	secretData.Database_host = string(pgSecret.Data["host"])
	secretData.Database_type = string(pgSecret.Data["type"])
	secretData.Database_sslmode = string(pgSecret.Data["sslmode"])

	if string(pgSecret.Data["type"]) != "" {
		secretData.Database_type = string(pgSecret.Data["type"])
	}

	secretSerialized, _ := yaml.Marshal(secretData)
	execCmd := []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + backupDir + "/secrets.yaml",
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup admin and postgres secrets")
		//return ctrl.Result{}, err
	}

	log.Info("Admin and postgres secrets backup finished")

	// FIELDS ENCRYPTION SECRET
	encSecret := encryptionFieldSecret{}
	fieldEncryptionSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-db-fields-encryption", Namespace: pulpBackup.Namespace}, fieldEncryptionSecret)
	if err != nil {
		log.Error(err, "Error trying to find pulp-db-fields-encryption secret")
		//return ctrl.Result{}, err
	}

	encSecret.Db_fields_encryption_key = string(fieldEncryptionSecret.Data["database_fields.symmetric.key"])
	encSecret.Db_fields_encryption_secret = pulpBackup.Spec.PulpInstanceName + "-db-fields-encryption"
	secretSerialized, _ = yaml.Marshal(encSecret)
	execCmd = []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + backupDir + "/db_fields_encryption_secret.yaml",
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup fields encryption secret")
		//return ctrl.Result{}, err
	}

	log.Info("Fields encryption secret backup finished")

	// SIGNING SECRET
	bkpSigningSecret := signingSecret{}
	signingConfig := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulp.Spec.SigningSecret, Namespace: pulpBackup.Namespace}, signingConfig)
	if err != nil {
		log.Error(err, "Error trying to find signing secret")
		//return ctrl.Result{}, err
	}
	bkpSigningSecret.Signing_secret = pulp.Spec.SigningSecret
	bkpSigningSecret.Signing_service_asc = string(signingConfig.Data["signing_service.asc"])
	bkpSigningSecret.Signing_service_gpg = string(signingConfig.Data["signing_service.gpg"])
	secretSerialized, _ = yaml.Marshal(bkpSigningSecret)
	execCmd = []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + backupDir + "/signing_secret.yaml",
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup signing secret")
		//return ctrl.Result{}, err
	}

	log.Info("Signing secret backup finished")

	// CONTAINER TOKEN SECRET
	bkpTokenSecret := containerTokenSecret{}
	tokenSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-container-auth", Namespace: pulpBackup.Namespace}, tokenSecret)
	if err != nil {
		log.Error(err, "Error trying to find container token secret")
		//return ctrl.Result{}, err
	}
	bkpTokenSecret.Container_token_secret = pulpBackup.Spec.PulpInstanceName + "-container-auth"
	bkpTokenSecret.Container_auth_public_key = string(tokenSecret.Data["container_auth_public_key.pem"])
	bkpTokenSecret.Container_auth_private_key = string(tokenSecret.Data["container_auth_private_key.pem"])
	secretSerialized, _ = yaml.Marshal(bkpTokenSecret)
	execCmd = []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + backupDir + "/container_token_secret.yaml",
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup container token secret")
		//return ctrl.Result{}, err
	}

	log.Info("Container token secret backup finished")

	// [WIP] OBJECT STORAGE S3 SECRET
	if pulp.Spec.ObjectStorageS3Secret != "" {
		log.Info("Backing up storage s3 secret ...")
	}

}
