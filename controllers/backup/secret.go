package pulp_backup

import (
	"context"

	"gopkg.in/yaml.v3"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// secretType contains all the information needed to make the backup of the secret
type secretType struct {

	// name of the key that will be used to store the secret name
	name string

	// PulpBackup instance
	pulpBackup *repomanagerv1alpha1.PulpBackup

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
func (r *PulpBackupReconciler) backupSecret(ctx context.Context, pulpBackup *repomanagerv1alpha1.PulpBackup, backupDir string, pod *corev1.Pod) error {
	log := ctrllog.FromContext(ctx)

	// we are considering that pulp CR instance is running in the same namespace as pulpbackup and
	// that there is only a single instance of pulp CR available
	// we could also let users pass the name of pulp instance
	pulp := &repomanagerv1alpha1.Pulp{}
	err := r.Get(ctx, types.NamespacedName{Name: pulpBackup.Spec.InstanceName, Namespace: pulpBackup.Namespace}, pulp)
	if err != nil {
		log.Error(err, "Failed to get PulpBackup")
		return err
	}

	// pulp-admin and pulp-postgres-configuration secrets will not be stored in secret.yaml file like in pulp-operator
	// we are splitting them in admin_secret.yaml and postgres_configuration.yaml files
	// PULP-ADMIN SECRET
	err = r.createBackupFile(ctx, secretType{"admin_password_secret", pulpBackup, backupDir, "admin_secret.yaml", pulpBackup.Spec.AdminPasswordSecret, pod})
	if err != nil {
		return err
	}
	log.Info("Admin secret backup finished")

	// POSTGRES SECRET (we are not following the same name for the keys that we defined in pulp-operator)
	err = r.createBackupFile(ctx, secretType{"postgres_secret", pulpBackup, backupDir, "postgres_configuration_secret.yaml", pulpBackup.Spec.InstanceName + "-postgres-configuration", pod})
	if err != nil {
		return err
	}
	log.Info("Postgres configuration secret backup finished")

	// FIELDS ENCRYPTION SECRET
	err = r.createBackupFile(ctx, secretType{"db_fields_encryption_secret", pulpBackup, backupDir, "container_token_secret.yaml", pulpBackup.Spec.InstanceName + "-db-fields-encryption", pod})
	if err != nil {
		return err
	}
	log.Info("Fields encryption secret backup finished")

	// SIGNING SECRET
	err = r.createBackupFile(ctx, secretType{"signing_secret", pulpBackup, backupDir, "signing_secret.yaml", pulp.Spec.SigningSecret, pod})
	if err != nil {
		return err
	}
	log.Info("Signing secret backup finished")

	// CONTAINER TOKEN SECRET
	err = r.createBackupFile(ctx, secretType{"container_token_secret", pulpBackup, backupDir, "db_fields_encryption_secret.yaml", pulpBackup.Spec.InstanceName + "-container-auth", pod})
	if err != nil {
		return err
	}
	log.Info("Container token secret backup finished")

	// OBJECT STORAGE S3 SECRET
	if len(pulp.Spec.ObjectStorageS3Secret) > 0 {
		err = r.createBackupFile(ctx, secretType{"storage_secret", pulpBackup, backupDir, "objectstorage_secret.yaml", pulp.Spec.ObjectStorageS3Secret, pod})
		if err != nil {
			return err
		}
		log.Info("Object storage s3 secret backup finished")
	}

	// OBJECT STORAGE AZURE SECRET
	if len(pulp.Spec.ObjectStorageAzureSecret) > 0 {
		err = r.createBackupFile(ctx, secretType{"storage_secret", pulpBackup, backupDir, "objectstorage_secret.yaml", pulp.Spec.ObjectStorageAzureSecret, pod})
		if err != nil {
			return err
		}
		log.Info("Object storage azure secret backup finished")
	}

	// OBJECT SSO CONFIG SECRET
	if len(pulp.Spec.SSOSecret) > 0 {
		err = r.createBackupFile(ctx, secretType{"sso_secret", pulpBackup, backupDir, "sso_secret.yaml", pulp.Spec.SSOSecret, pod})
		if err != nil {
			return err
		}
		log.Info("SSO secret backup finished")
	}

	return nil
}

// createBackupFile stores the content of the secrets in a file located in a backup PV
func (r *PulpBackupReconciler) createBackupFile(ctx context.Context, secretType secretType) error {
	log := ctrllog.FromContext(ctx)

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
	_, err = r.containerExec(secretType.pod, execCmd, secretType.pulpBackup.Name+"-backup-manager", secretType.pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to backup "+secretType.secretName+" secret")
		return err
	}

	log.Info("Container token secret backup finished")
	return nil
}
