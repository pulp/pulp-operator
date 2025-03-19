package repo_manager_backup

import (
	"bytes"
	"context"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
)

// secretType contains all the information needed to make the backup of the secret
type secretType struct {

	// name of the key that will be used to store the secret name
	name string

	// PulpBackup instance
	pulpBackup *pulpv1.PulpBackup

	// path of where the backup will be stored (PVC mount point)
	backupDir string

	// name of the backup file
	backupFile string

	// name of the secret that will be copied
	secretName string

	// backup-manager pod
	pod *corev1.Pod
}

// backupSecrets makes a copy of the Secrets used by Pulp components
func (r *RepoManagerBackupReconciler) backupSecret(ctx context.Context, pulpBackup *pulpv1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := r.RawLogger
	deploymentName := getDeploymentName(pulpBackup)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &pulpv1.Pulp{}
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: pulpBackup.Namespace}, pulp); err != nil {
		log.Error(err, "Failed to get Pulp")
		return err
	}

	pulpSecretKey := getPulpSecretKey(pulpBackup)
	adminPasswordSecret := getAdminPasswordSecret(pulpBackup)
	postgresCfgSecret := getPostgresCfgSecret(pulpBackup)
	dbFieldsEncryption := getDBFieldsEncryption(pulp)
	containerTokenSecret := getContainerTokenSecret(pulp)

	// PULP-SECRET-KEY
	if err := r.createSecretBackupFile(ctx, secretType{"pulp_secret_key", pulpBackup, backupDir, "pulp_secret_key.yaml", pulpSecretKey, pod}); err != nil {
		return err
	}
	log.Info("PulpSecretKey secret backup finished")

	// pulp-admin and pulp-postgres-configuration secrets will not be stored in secret.yaml file like in pulp-operator
	// we are splitting them in admin_secret.yaml and postgres_configuration.yaml files
	// PULP-ADMIN SECRET
	if err := r.createBackupFile(ctx, secretType{"admin_password_secret", pulpBackup, backupDir, "admin_secret.yaml", adminPasswordSecret, pod}); err != nil {
		return err
	}
	log.Info("Admin secret backup finished")

	// POSTGRES SECRET (we are not following the same name for the keys that we defined in pulp-operator)
	if err := r.createBackupFile(ctx, secretType{"postgres_secret", pulpBackup, backupDir, "postgres_configuration_secret.yaml", postgresCfgSecret, pod}); err != nil {
		return err
	}
	log.Info("Postgres configuration secret backup finished")

	// FIELDS ENCRYPTION SECRET
	if err := r.createBackupFile(ctx, secretType{"db_fields_encryption_secret", pulpBackup, backupDir, "db_fields_encryption_secret.yaml", dbFieldsEncryption, pod}); err != nil {
		return err
	}
	log.Info("Fields encryption secret backup finished")

	// SIGNING SECRET
	if len(pulp.Spec.SigningSecret) > 0 {
		if err := r.createBackupFile(ctx, secretType{"signing_secret", pulpBackup, backupDir, "signing_secret.yaml", pulp.Spec.SigningSecret, pod}); err != nil {
			return err
		}
		log.Info("Signing secret backup finished")
		if err := r.createSecretBackupFile(ctx, secretType{"signing_scripts", pulpBackup, backupDir, "signing_scripts.yaml", pulp.Spec.SigningScripts, pod}); err != nil {
			return err
		}
	}

	// CONTAINER TOKEN SECRET
	if err := r.createBackupFile(ctx, secretType{"container_token_secret", pulpBackup, backupDir, "container_token_secret.yaml", containerTokenSecret, pod}); err != nil {
		return err
	}
	log.Info("Container token secret backup finished")

	// OBJECT STORAGE S3 SECRET
	if len(pulp.Spec.ObjectStorageS3Secret) > 0 {
		if err := r.createBackupFile(ctx, secretType{"storage_secret", pulpBackup, backupDir, "objectstorage_secret.yaml", pulp.Spec.ObjectStorageS3Secret, pod}); err != nil {
			return err
		}
		log.Info("Object storage s3 secret backup finished")
	}

	// OBJECT STORAGE AZURE SECRET
	if len(pulp.Spec.ObjectStorageAzureSecret) > 0 {
		if err := r.createBackupFile(ctx, secretType{"storage_secret", pulpBackup, backupDir, "objectstorage_secret.yaml", pulp.Spec.ObjectStorageAzureSecret, pod}); err != nil {
			return err
		}
		log.Info("Object storage azure secret backup finished")
	}

	// OBJECT SSO CONFIG SECRET
	if len(pulp.Spec.SSOSecret) > 0 {
		if err := r.createBackupFile(ctx, secretType{"sso_secret", pulpBackup, backupDir, "sso_secret.yaml", pulp.Spec.SSOSecret, pod}); err != nil {
			return err
		}
		log.Info("SSO secret backup finished")
	}

	// LDAP CONFIG SECRET
	if len(pulp.Spec.LDAP.Config) > 0 {
		if err := r.createSecretBackupFile(ctx, secretType{"ldap_secret", pulpBackup, backupDir, "ldap_secret.yaml", pulp.Spec.LDAP.Config, pod}); err != nil {
			return err
		}
		log.Info("LDAP secret backup finished")
	}
	// LDAP CA SECRET
	if len(pulp.Spec.LDAP.CA) > 0 {
		if err := r.createSecretBackupFile(ctx, secretType{"ldap_ca_secret", pulpBackup, backupDir, "ldap_ca_secret.yaml", pulp.Spec.LDAP.CA, pod}); err != nil {
			return err
		}
		log.Info("LDAP CA secret backup finished")
	}

	return nil
}

// createBackupFile stores the content of the secrets in a file located in a backup PV
func (r *RepoManagerBackupReconciler) createBackupFile(ctx context.Context, secretType secretType) error {
	log := r.RawLogger

	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretType.secretName, Namespace: secretType.pulpBackup.Namespace}, secret)
	if err != nil {
		log.Error(err, "Error trying to find "+secretType.secretName+" secret")
		return err
	}

	bkpContent := map[string]string{}
	bkpContent[secretType.name] = secretType.secretName
	for key, value := range secret.Data {
		bkpContent[key] = string(value)
	}

	var secretSerialized []byte
	secretSerialized, _ = yaml.Marshal(bkpContent)

	execCmd := []string{
		"bash", "-c", "echo '" + string(secretSerialized) + "' > " + secretType.backupDir + "/" + secretType.backupFile,
	}
	_, err = controllers.ContainerExec(ctx, r, secretType.pod, execCmd, secretType.pulpBackup.Name+"-backup-manager", secretType.pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+secretType.secretName+" secret")
		return err
	}

	log.Info("Container token secret backup finished")
	return nil
}

// createSecretBackupFile stores a copy of the Secrets in YAML format.
// Since we don't need to keep compatibility with ansible version anymore, this
// method does not need to follow an specific struct and should work with any Secret.
func (r *RepoManagerBackupReconciler) createSecretBackupFile(ctx context.Context, secretType secretType) error {
	log := r.RawLogger
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretType.secretName, Namespace: secretType.pulpBackup.Namespace}, secret)
	if err != nil {
		log.Error(err, "Error trying to find "+secretType.secretName+" secret")
		return err
	}

	secretYaml := new(bytes.Buffer)
	ymlPrinter := printers.YAMLPrinter{}
	ymlPrinter.PrintObj(secret, secretYaml)

	execCmd := []string{
		"bash", "-c", "echo '" + secretYaml.String() + "' > " + secretType.backupDir + "/" + secretType.backupFile,
	}
	_, err = controllers.ContainerExec(ctx, r, secretType.pod, execCmd, secretType.pulpBackup.Name+"-backup-manager", secretType.pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+secretType.secretName+" secret")
		return err
	}

	log.Info("Secret " + secretType.secretName + " backup finished")
	return nil
}
