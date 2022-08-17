package pulp_restore

import (
	"context"
	"reflect"

	repomanagerv1alpha1 "github.com/git-hyagi/pulp-operator-go/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type adminPassword struct {
	AdminPasswordSecret string `json:"admin_password_secret"`
	Password            string `json:"password"`
}

type postgresSecret struct {
	Database        string `json:"database"`
	Host            string `json:"host"`
	Password        string `json:"password"`
	Port            string `json:"port"`
	Postgres_secret string `json:"postgres_secret"`
	Sslmode         string `json:"sslmode"`
	Type            string `json:"type"`
	Username        string `json:"username"`
}

type containerTokenSecret struct {
	DatabaseFields           string `json:"database_fields.symmetric.key"`
	DbFieldsEncryptionSecret string `json:"db_fields_encryption_secret"`
}

type storageObjectSecret struct {
	S3AccessKeyId         string `json:"s3-access-key-id"`
	S3BucketName          string `json:"s3-bucket-name"`
	S3Region              string `json:"s3-region"`
	S3SecretAccessKey     string `json:"s3-secret-access-key"`
	StorageSecret         string `json:"storage_secret"`
	AzureAccountName      string `json:"azure-account-name"`
	AzureAccountKey       string `json:"azure-account-key"`
	AzureContainer        string `json:"azure-container"`
	AzureContainerPath    string `json:"azure-container-path"`
	AzureConnectionString string `json:"azure-connection-string"`
}

type signingSecret struct {
	SigninSecret      string `json:"signing_secret"`
	SigninServiceASC  string `json:"signing_service.asc"`
	SigningServiceGPG string `json:"signing_service.gpg"`
}

type ssoSecret struct {
	KeycloakAdminRole           string `json:"keycloak_admin_role"`
	KeycloakGroupTokenClaim     string `json:"keycloak_group_token_claim"`
	KeycloakHost                string `json:"keycloak_host"`
	KeycloakPort                string `json:"keycloak_port"`
	KeycloakProtocol            string `json:"keycloak_protocol"`
	KeycloakRealm               string `json:"keycloak_realm"`
	SocialAuthKeycloakKey       string `json:"social_auth_keycloak_key"`
	SocialAuthKeycloakPublicKey string `json:"social_auth_keycloak_public_key"`
	SocialAuthKeycloakSecret    string `json:"social_auth_keycloak_secret"`
	SSOSecret                   string `json:"sso_secret"`
}

const (
	resourceTypeAdminPassword  = "AdminPassword"
	resourceTypePostgres       = "Postgres"
	resourceTypeObjectStorage  = "ObjectStorage"
	resourceTypeSigningSecret  = "Signing"
	resourceTypeContainerToken = "ContainerToken"
)

// restoreSecret restores the operator secrets created by pulpbackup CR
func (r *PulpRestoreReconciler) restoreSecret(ctx context.Context, pulpRestore *repomanagerv1alpha1.PulpRestore, backupDir string, pod *corev1.Pod) error {

	// restore admin password secret
	if err := r.secret(ctx, resourceTypeAdminPassword, "admin_password_secret", backupDir, "admin_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore postgres secret
	if err := r.secret(ctx, resourceTypePostgres, "postgres_secret", backupDir, "postgres_configuration_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore object storage secret
	if err := r.secret(ctx, resourceTypeObjectStorage, "storage_secret", backupDir, "objectstorage_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore container token secret
	if err := r.secret(ctx, resourceTypeContainerToken, "db_fields_encryption_secret", backupDir, "container_token_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore signing secret
	if err := r.secret(ctx, resourceTypeSigningSecret, "signing_secret", backupDir, "signing_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore sso secret
	if err := r.secret(ctx, resourceTypeSigningSecret, "sso_secret", backupDir, "sso_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	return nil
}

// secret creates the secret k8s resource from the backup file (backupFile) based on
// resourceType: the type of the secret (like AdminPassword, or ObjectStorage, or ContainerToken, etc)
// secretNameKey: is the secret's key that contains the secret name to be restored
func (r *PulpRestoreReconciler) secret(ctx context.Context, resourceType, secretNameKey, backupDir, backupFile string, pod *corev1.Pod, pulpRestore *repomanagerv1alpha1.PulpRestore) error {
	log := ctrllog.FromContext(ctx)

	secretNameData := ""
	execCmd := []string{
		"test", "-f", backupDir + "/" + backupFile,
	}
	_, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		return err
	} else { // if backupFile file found

		// retrieving backup file content
		log.Info("Restoring " + resourceType + " secret ...")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restoring "+resourceType+" secret", "Restoring"+resourceType+"Secret")
		execCmd = []string{
			"cat", backupDir + "/" + backupFile,
		}
		cmdOutput, err := r.containerExec(pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to get "+backupFile+"!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get "+backupFile, "FailedGet"+resourceType+"Secret")
			return err
		}

		// "assert" struct type based on secretNameKey
		secretData := map[string]string{}
		var v reflect.Value
		switch secretNameKey {
		case "signing_secret":
			secretType := signingSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "db_fields_encryption_secret":
			secretType := containerTokenSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "storage_secret":
			secretType := storageObjectSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "postgres_secret":
			secretType := postgresSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "admin_password_secret":
			secretType := adminPassword{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "sso_secret":
			secretType := ssoSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		}

		// loop through all fields from struct
		for i := 0; i < v.NumField(); i++ {

			// if struct field tag ("alias") is the same as secretNameKey
			// we will keep the field content instead of storing it in
			// secretData because it should not be part of the secret data itself
			if v.Type().Field(i).Tag.Get("json") == secretNameKey {
				secretNameData = v.Field(i).String()
				continue
			}

			// if the field is not empty("") we are getting the values and
			// storing it in secretData
			if v.Field(i).String() != "" {
				secretData[v.Type().Field(i).Tag.Get("json")] = v.Field(i).String()
			}
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretNameData,
				Namespace: pulpRestore.Namespace,
			},
			StringData: secretData,
		}
		err = r.Create(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to create "+resourceType+" secret!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Error trying to restore "+resourceType+" secret!", "FailedCreate"+resourceType+"Secret")
			return err
		}
		log.Info(resourceType + " secret restored")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", resourceType+" secret restored", resourceType+"SecretRestored")
	}

	return nil
}
