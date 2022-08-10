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

func (r *PulpBackupReconciler) backupSecret(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	// ADMIN AND POSTGRES SECRETS
	err := r.createSecret(ctx, pulpBackup, backupDir, pod, "admin_and_postgres")
	if err != nil {
		return err
	}
	log.Info("Admin and postgres secrets backup finished")

	// FIELDS ENCRYPTION SECRET
	err = r.createSecret(ctx, pulpBackup, backupDir, pod, "fields_encryption_secret")
	if err != nil {
		return err
	}
	log.Info("Fields encryption secret backup finished")

	// SIGNING SECRET
	err = r.createSecret(ctx, pulpBackup, backupDir, pod, "signing_secret")
	if err != nil {
		return err
	}
	log.Info("Signing secret backup finished")

	// CONTAINER TOKEN SECRET
	err = r.createSecret(ctx, pulpBackup, backupDir, pod, "container_token")
	if err != nil {
		return err
	}
	log.Info("Container token secret backup finished")

	// [WIP] OBJECT STORAGE S3 SECRET
	//r.createSecret(ctx, pulpBackup, backupDir, pod, "s3_secret")

	// [WIP] OBJECT STORAGE AZURE SECRET
	//r.createSecret(ctx, pulpBackup, backupDir, pod, "azure_secret")

	// [WIP] OBJECT SSO CONFIG SECRET
	//r.createSecret(ctx, pulpBackup, backupDir, pod, "sso_config_secret")

	return nil
}

func (r *PulpBackupReconciler) createSecret(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod, secretType string) error {
	log := ctrllog.FromContext(ctx)

	var secretSerialized []byte
	backupFile := ""
	secret := &corev1.Secret{}

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get PulpBackup")
		return err
	}

	switch secretType {
	case "admin_and_postgres":
		err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.AdminPasswordSecret, Namespace: pulpBackup.Namespace}, secret)
		if err != nil {
			log.Error(err, "Error trying to find pulp-admin-password secret")
			return err
		}

		adminPwdSecret := adminSecret{
			Admin_password_name: pulpBackup.Spec.AdminPasswordSecret,
			Admin_password:      string(secret.Data["password"]),
		}

		err = r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-postgres-configuration", Namespace: pulpBackup.Namespace}, secret)
		if err != nil {
			log.Error(err, "Error trying to find pulp-postgres-configuration secret")
			return err
		}

		adminPwdSecret.Database_password = string(secret.Data["password"])
		adminPwdSecret.Database_username = string(secret.Data["username"])
		adminPwdSecret.Database_name = string(secret.Data["database"])
		adminPwdSecret.Database_port = string(secret.Data["port"])
		adminPwdSecret.Database_host = string(secret.Data["host"])
		adminPwdSecret.Database_type = string(secret.Data["type"])
		adminPwdSecret.Database_sslmode = string(secret.Data["sslmode"])

		if string(secret.Data["type"]) != "" {
			adminPwdSecret.Database_type = string(secret.Data["type"])
		}

		secretSerialized, _ = yaml.Marshal(adminPwdSecret)
		backupFile = "secrets.yaml"

	case "container_token":
		err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-container-auth", Namespace: pulpBackup.Namespace}, secret)
		if err != nil {
			log.Error(err, "Error trying to find container token secret")
			return err
		}

		bkpTokenSecret := containerTokenSecret{
			Container_token_secret:     pulpBackup.Spec.PulpInstanceName + "-container-auth",
			Container_auth_public_key:  string(secret.Data["container_auth_public_key.pem"]),
			Container_auth_private_key: string(secret.Data["container_auth_private_key.pem"]),
		}
		secretSerialized, _ = yaml.Marshal(bkpTokenSecret)
		backupFile = "db_fields_encryption_secret.yaml"

	case "signing_secret":
		err := r.Get(ctx, types.NamespacedName{Name: pulp.Spec.SigningSecret, Namespace: pulpBackup.Namespace}, secret)
		if err != nil {
			log.Error(err, "Error trying to find singing-secret secret")
			return err
		}

		bkpSigningSecret := signingSecret{
			Signing_secret:      pulp.Spec.SigningSecret,
			Signing_service_asc: string(secret.Data["signing_service.asc"]),
			Signing_service_gpg: string(secret.Data["signing_service.gpg"]),
		}
		secretSerialized, _ = yaml.Marshal(bkpSigningSecret)
		backupFile = "signing_secret.yaml"

	case "fields_encryption_secret":
		err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.PulpInstanceName + "-db-fields-encryption", Namespace: pulpBackup.Namespace}, secret)
		if err != nil {
			log.Error(err, "Error trying to find pulp-db-fields-encryption secret")
			return err
		}

		encSecret := encryptionFieldSecret{
			Db_fields_encryption_key:    string(secret.Data["database_fields.symmetric.key"]),
			Db_fields_encryption_secret: pulpBackup.Spec.PulpInstanceName + "-db-fields-encryption",
		}
		secretSerialized, _ = yaml.Marshal(encSecret)
		backupFile = "container_token_secret.yaml"

	case "s3_secret":
		if pulp.Spec.ObjectStorageS3Secret != "" {
			log.Info("Backing up storage s3 secret ...")
		}
	case "azure_secret":
		if pulp.Spec.ObjectStorageAzureSecret != "" {
			log.Info("Backing up storage azure secret ...")
		}
	case "sso_config_secret":
		if pulp.Spec.SSOSecret != "" {
			log.Info("Backing up sso configuration secret ...")
		}
	}

	execCmd := []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + backupDir + "/" + backupFile,
	}
	_, err = r.containerExec(pod, execCmd, pulpBackup.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+secretType+" secret")
		return err
	}

	log.Info("Container token secret backup finished")
	return nil
}
