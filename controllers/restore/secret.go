package repo_manager_restore

import (
	"context"
	"reflect"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

type adminPassword struct {
	AdminPasswordSecret string `json:"admin_password_secret"`
	Password            string `json:"password"`
}

type postgresSecret struct {
	Database       string `json:"database"`
	Host           string `json:"host"`
	Password       string `json:"password"`
	Port           string `json:"port"`
	PostgresSecret string `json:"postgres_secret"`
	Sslmode        string `json:"sslmode"`
	Type           string `json:"type"`
	Username       string `json:"username"`
}

type containerTokenSecret struct {
	ContainerAuthPublicKey  string `json:"container_auth_public_key.pem"`
	ContainerAuthPrivateKey string `json:"container_auth_private_key.pem"`
	ContainerTokenSecret    string `json:"container_token_secret"`
}

type dbFieldsEncryptionSecret struct {
	DatabaseFields           string `json:"database_fields.symmetric.key"`
	DbFieldsEncryptionSecret string `json:"db_fields_encryption_secret"`
}

type storageObjectSecret struct {
	S3AccessKeyId         string `json:"s3-access-key-id"`
	S3BucketName          string `json:"s3-bucket-name"`
	S3Region              string `json:"s3-region"`
	S3Endpoint            string `json:"s3-endpoint"`
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

type pulpSecretKey struct {
	PulpSecretKey string `json:"pulp_secret_key"`
	SecretKey     string `json:"secret_key"`
}

const (
	resourceTypePulpSecretKey      = "PulpSecreyKey"
	resourceTypeAdminPassword      = "AdminPassword"
	resourceTypePostgres           = "Postgres"
	resourceTypeObjectStorage      = "ObjectStorage"
	resourceTypeSigningSecret      = "Signing"
	resourceTypeSigningScripts     = "SigningScripts"
	resourceTypeContainerToken     = "ContainerToken"
	resourceTypeSSOSecret          = "SSO"
	resourceTypeDBFieldsEncryption = "DBFieldsEncryption"
	resourceTypeLDAP               = "LDAPSecret"
)

// restoreSecret restores the operator secrets created by pulpbackup CR
func (r *RepoManagerRestoreReconciler) restoreSecret(ctx context.Context, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore, backupDir string, pod *corev1.Pod) error {

	// [TODO]
	// type secretTypes struct {resourceType string, secretNameKey string, backupFile string}
	// secrets := []secretTypes{ ... }
	// for i in range secrets { r.secret(...) }

	r.RawLogger.V(1).Info("Restoring from golang backup version")

	// restore pulp-secret-key secret
	if _, err := r.restoreSecretFromYaml(ctx, resourceTypePulpSecretKey, "secret_key", backupDir, "pulp_secret_key.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore admin password secret
	if _, err := r.secret(ctx, resourceTypeAdminPassword, "admin_password_secret", backupDir, "admin_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore postgres secret
	if _, err := r.secret(ctx, resourceTypePostgres, "postgres_secret", backupDir, "postgres_configuration_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore db fields encryption secret
	if _, err := r.secret(ctx, resourceTypeDBFieldsEncryption, "db_fields_encryption_secret", backupDir, "db_fields_encryption_secret.yaml", pod, pulpRestore); err != nil {
		return err
	}

	// restore container token secret
	// this secret is not mandatory. If the backup file is not found is not an error
	if found, err := r.secret(ctx, resourceTypeContainerToken, "container_token_secret", backupDir, "container_token_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}

	// restore object storage secret
	// this secret is not mandatory. If the backup file is not found is not an error
	if found, err := r.secret(ctx, resourceTypeObjectStorage, "storage_secret", backupDir, "objectstorage_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}

	// restore signing secret
	// this secret is not mandatory. If the backup file is not found is not an error
	if found, err := r.secret(ctx, resourceTypeSigningSecret, "signing_secret", backupDir, "signing_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}
	if found, err := r.restoreSecretFromYaml(ctx, resourceTypeSigningScripts, "signing_scripts", backupDir, "signing_scripts.yaml", pod, pulpRestore); found && err != nil {
		return err
	}

	// restore sso secret
	// this secret is not mandatory. If the backup file is not found is not an error
	if found, err := r.secret(ctx, resourceTypeSSOSecret, "sso_secret", backupDir, "sso_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}

	// restore ldap secret(s)
	if found, err := r.restoreSecretFromYaml(ctx, resourceTypeLDAP, "ldap_secret", backupDir, "ldap_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}
	if found, err := r.restoreSecretFromYaml(ctx, resourceTypeLDAP, "ldap_ca_secret", backupDir, "ldap_ca_secret.yaml", pod, pulpRestore); found && err != nil {
		return err
	}

	return nil
}

// secret creates the secret k8s resource from the backup file (backupFile) based on
// resourceType: the type of the secret (like AdminPassword, or ObjectStorage, or ContainerToken, etc)
// secretNameKey: is the secret's key that contains the secret name to be restored
// it returns false and the error if the file is not found
func (r *RepoManagerRestoreReconciler) secret(ctx context.Context, resourceType, secretNameKey, backupDir, backupFile string, pod *corev1.Pod, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) (bool, error) {

	log := r.RawLogger

	secretNameData := ""
	execCmd := []string{
		"test", "-f", backupDir + "/" + backupFile,
	}
	_, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)

	// if backupFile file is not found return the error
	if err != nil {
		return false, err
	} else {
		// retrieving backup file content
		log.Info("Restoring " + resourceType + " secret ...")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Restoring "+resourceType+" secret", "Restoring"+resourceType+"Secret")
		execCmd = []string{
			"cat", backupDir + "/" + backupFile,
		}
		cmdOutput, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
		if err != nil {
			log.Error(err, "Failed to get "+backupFile+"!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get "+backupFile, "FailedGet"+resourceType+"Secret")
			return true, err
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
			secretType := dbFieldsEncryptionSecret{}
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
		case "container_token_secret":
			secretType := containerTokenSecret{}
			yaml.Unmarshal([]byte(cmdOutput), &secretType)
			v = reflect.ValueOf(secretType)
		case "pulp_secret_key":
			secretType := pulpSecretKey{}
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
				setStatusField(secretNameKey, v.Field(i).String(), pulpRestore)
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

		// we'll recreate the secret only if it was not found
		// in situations like during a pulpRestore reconcile loop (because of an error) the secret could have been previously created
		// this will avoid an infinite reconciliation loop trying to recreate a resource that already exists
		if err := r.Get(ctx, types.NamespacedName{Name: secretNameData, Namespace: pulpRestore.Namespace}, secret); err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, secret); err != nil {
				log.Error(err, "Failed to create "+resourceType+" secret!")
				r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Error trying to restore "+resourceType+" secret!", "FailedCreate"+resourceType+"Secret")
				return true, err
			}
			log.Info(resourceType + " secret restored")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", resourceType+" secret restored", resourceType+"SecretRestored")
		}
	}

	return true, nil
}

// setStatusField sets the pulpRestore.Status.FieldName with fieldValue
func setStatusField(fieldName, fieldValue string, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) error {

	s := reflect.ValueOf(pulpRestore.Status)
	// iterate over the fields from pulpRestore.Status struct
	for i := 0; i < s.NumField(); i++ {

		// if this is the field that we want to update
		if s.Type().Field(i).Tag.Get("json") == fieldName {

			// set the new value of .Status.fieldName with fieldValue
			// we need to pass a pointer to pulpRestore.Status because we want to modify
			// its contents
			reflect.ValueOf(&pulpRestore.Status).Elem().Field(i).SetString(fieldValue)
			break
		}
	}

	return nil
}

// restoreSecretFromYaml restores the Secret from a YAML file.
// Since we don't need to keep compatibility with ansible version anymore, this
// method does not need to follow an specific struct and should work with any Secret.
func (r *RepoManagerRestoreReconciler) restoreSecretFromYaml(ctx context.Context, resourceType, secretNameKey, backupDir, backupFile string, pod *corev1.Pod, pulpRestore *repomanagerpulpprojectorgv1beta2.PulpRestore) (bool, error) {

	log := r.RawLogger
	ldapSecretFile := backupDir + "/" + backupFile
	execCmd := []string{
		"test", "-f", ldapSecretFile,
	}
	_, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)

	// if no ldap secret found there is nothing to be restored
	if err != nil {
		return false, nil
	}
	execCmd = []string{
		"cat", ldapSecretFile,
	}
	cmdOutput, err := controllers.ContainerExec(r, pod, execCmd, pulpRestore.Name+"-backup-manager", pod.Namespace)
	if err != nil {
		log.Error(err, "Failed to get "+backupFile+"!")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Failed to get "+backupFile, "FailedGet"+resourceType+"Secret")
		return true, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, _ := decode([]byte(cmdOutput), nil, nil)
	secret := obj.(*corev1.Secret)

	// "removing" fields from backup to avoid errors
	secret.ObjectMeta.ResourceVersion = ""
	secret.ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}

	// we'll recreate the secret only if it was not found
	// in situations like during a pulpRestore reconcile loop (because of an error) the secret could have been previously created
	// this will avoid an infinite reconciliation loop trying to recreate a resource that already exists
	if err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: pulpRestore.Namespace}, secret); err != nil && errors.IsNotFound(err) {
		if err := r.Create(ctx, secret); err != nil {
			log.Error(err, "Failed to create "+resourceType+" secret!")
			r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", "Error trying to restore "+resourceType+" secret!", "FailedCreate"+resourceType+"Secret")
			return true, err
		}
		log.Info(resourceType + " secret restored")
		r.updateStatus(ctx, pulpRestore, metav1.ConditionFalse, "RestoreComplete", resourceType+" secret restored", resourceType+"SecretRestored")
	}

	return true, nil
}
