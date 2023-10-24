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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// [TODO] move the pending secrets tasks (for ex, create/reconcile) from api.go
// to here. Since there is no need to keep the same "struct" as of in ansible
// version, we can now do a better organization of the resources.

// pulpServerSecret creates the pulp-server secret object which is used to
// populate the /etc/pulp/settings.py config file
func pulpServerSecret(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	pulp_settings := ""

	// default settings.py configuration
	defaultPulpSettings(resources, &pulp_settings)

	// db settings
	databaseSettings(resources, &pulp_settings)

	// add cache settings
	cacheSettings(resources, &pulp_settings)

	// azure settings
	azureSettings(resources, &pulp_settings)

	// s3 settings
	s3Settings(resources, &pulp_settings)

	// configure settings.py with keycloak integration variables
	ssoConfig(resources, &pulp_settings)

	// configure TOKEN_SERVER based on ingress_type
	tokenSettings(resources, &pulp_settings)

	// django SECRET_KEY
	secretKeySettings(resources, &pulp_settings)

	// allowed content checksum
	allowedContentChecksumsSettings(resources, &pulp_settings)

	// ldap auth config
	ldapSettings(resources, &pulp_settings)

	// add custom settings to the secret
	addCustomPulpSettings(pulp, &pulp_settings)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.PulpServerSecret(pulp.Name),
			Namespace: pulp.Namespace,
			Labels:    settings.CommonLabels(*pulp),
		},
		StringData: map[string]string{
			"settings.py": pulp_settings,
		},
	}

	// Set Pulp instance as the owner and controller
	ctrl.SetControllerReference(pulp, sec, resources.Scheme)
	return sec
}

// pulp-db-fields-encryption secret
func pulpDBFieldsEncryptionSecret(resources controllers.FunctionResources) client.Object {
	pulp := resources.Pulp
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulp.Spec.DBFieldsEncryptionSecret,
			Namespace: pulp.Namespace,
			Labels:    settings.CommonLabels(*pulp),
		},
		StringData: map[string]string{
			"database_fields.symmetric.key": createFernetKey(),
		},
	}
	return sec
}

// pulp-admin-password
func pulpAdminPasswordSecret(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulp.Spec.AdminPasswordSecret,
			Namespace: pulp.Namespace,
			Labels:    settings.CommonLabels(*pulp),
		},
		StringData: map[string]string{
			"password": createPwd(32),
		},
	}
	ctrl.SetControllerReference(pulp, sec, resources.Scheme)

	return sec
}

// pulpDjangoKeySecret defines the Secret with the pulp-secret-key
func pulpDjangoKeySecret(resources controllers.FunctionResources) client.Object {
	pulp := resources.Pulp
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulp.Spec.PulpSecretKey,
			Namespace: pulp.Namespace,
			Labels:    settings.CommonLabels(*pulp),
		},
		StringData: map[string]string{
			"secret_key": djangoKey(),
		},
	}
	ctrl.SetControllerReference(pulp, sec, resources.Scheme)
	return sec
}

// pulp-container-auth
func pulpContainerAuth(resources controllers.FunctionResources) client.Object {
	pulp := resources.Pulp
	privKey, pubKey := genTokenAuthKey()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pulp.Spec.ContainerTokenSecret,
			Namespace: pulp.Namespace,
			Labels:    settings.CommonLabels(*pulp),
		},
		StringData: map[string]string{
			"container_auth_private_key.pem": privKey,
			"container_auth_public_key.pem":  pubKey,
		},
	}
}

// defaultPulpSettings appends some common settings into pulpSettings
func defaultPulpSettings(resources controllers.FunctionResources, pulpSettings *string) {
	rootUrl := getRootURL(resources)
	*pulpSettings = *pulpSettings + controllers.DotNotEditMessage + `
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
GALAXY_CONTAINER_SIGNING_SERVICE = "container-default"
ANSIBLE_API_HOSTNAME = "` + rootUrl + `"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys/"
CONTENT_ORIGIN = "` + rootUrl + `"
GALAXY_FEATURE_FLAGS = {
  'execution_environments': 'True',
}
PRIVATE_KEY_PATH = "/etc/pulp/keys/container_auth_private_key.pem"
PUBLIC_KEY_PATH = "/etc/pulp/keys/container_auth_public_key.pem"
STATIC_ROOT = "/var/lib/operator/static/"
TOKEN_AUTH_DISABLED = False
TOKEN_SIGNATURE_ALGORITHM = "ES256"
API_ROOT = "/pulp/"
`
}

// cacheSettings appends redis/cache settings into pulpSettings
func cacheSettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	context := resources.Context
	client := resources.Client

	if !pulp.Spec.Cache.Enabled {
		return
	}

	var cacheHost, cachePort, cachePassword, cacheDB string

	cachePort = strconv.Itoa(6379)
	if pulp.Spec.Cache.RedisPort != 0 {
		cachePort = strconv.Itoa(pulp.Spec.Cache.RedisPort)
	}
	cacheHost = pulp.Name + "-redis-svc." + pulp.Namespace
	if len(pulp.Spec.Cache.ExternalCacheSecret) > 0 {
		// retrieve the connection data from ExternalCacheSecret secret
		externalCacheData := []string{"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB"}
		externalCacheConfig, _ := controllers.RetrieveSecretData(context, pulp.Spec.Cache.ExternalCacheSecret, pulp.Namespace, true, client, externalCacheData...)
		cacheHost = externalCacheConfig["REDIS_HOST"]
		cachePort = externalCacheConfig["REDIS_PORT"]
		cachePassword = externalCacheConfig["REDIS_PASSWORD"]
		cacheDB = externalCacheConfig["REDIS_DB"]
	}

	*pulpSettings = *pulpSettings + `CACHE_ENABLED = True
REDIS_HOST =  "` + cacheHost + `"
REDIS_PORT =  "` + cachePort + `"
REDIS_PASSWORD = "` + cachePassword + `"
REDIS_DB = "` + cacheDB + `"
`
}

// databaseSettings appends postgres settings into pulpSettings
func databaseSettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	logger := resources.Logger
	context := resources.Context
	client := resources.Client

	var dbHost, dbPort, dbUser, dbPass, dbName, dbSSLMode string

	// if there is no external database configuration get the databaseconfig from pulp-postgres-configuration secret
	if len(pulp.Spec.Database.ExternalDBSecret) == 0 {
		postgresConfigurationSecret := pulp.Name + "-postgres-configuration"

		logger.V(1).Info("Retrieving Postgres credentials from "+postgresConfigurationSecret+" secret", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		pgCredentials, err := controllers.RetrieveSecretData(context, postgresConfigurationSecret, pulp.Namespace, true, client, "username", "password", "database", "port", "sslmode")
		if err != nil {
			logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Name)
			return
		}
		dbHost = pulp.Name + "-database-svc"
		dbPort = pgCredentials["port"]
		dbUser = pgCredentials["username"]
		dbPass = pgCredentials["password"]
		dbName = pgCredentials["database"]
		dbSSLMode = pgCredentials["sslmode"]
	} else {
		logger.V(1).Info("Retrieving Postgres credentials from "+resources.Pulp.Spec.Database.ExternalDBSecret+" secret", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Name)
		externalPostgresData := []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USERNAME", "POSTGRES_PASSWORD", "POSTGRES_DB_NAME", "POSTGRES_SSLMODE"}
		pgCredentials, err := controllers.RetrieveSecretData(context, pulp.Spec.Database.ExternalDBSecret, pulp.Namespace, true, client, externalPostgresData...)
		if err != nil {
			logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Name)
			return
		}
		dbHost = pgCredentials["POSTGRES_HOST"]
		dbPort = pgCredentials["POSTGRES_PORT"]
		dbUser = pgCredentials["POSTGRES_USERNAME"]
		dbPass = pgCredentials["POSTGRES_PASSWORD"]
		dbName = pgCredentials["POSTGRES_DB_NAME"]
		dbSSLMode = pgCredentials["POSTGRES_SSLMODE"]
	}

	*pulpSettings = *pulpSettings + `DATABASES = {
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
`
}

// azureSettings appends azure blob object storage settings into pulpSettings
func azureSettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	logger := resources.Logger
	context := resources.Context
	client := resources.Client

	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	if storageType[0] != controllers.AzureObjType {
		return
	}

	logger.V(1).Info("Retrieving Azure data from " + resources.Pulp.Spec.ObjectStorageAzureSecret)
	storageData, err := controllers.RetrieveSecretData(context, pulp.Spec.ObjectStorageAzureSecret, pulp.Namespace, true, client, "azure-account-name", "azure-account-key", "azure-container", "azure-container-path", "azure-connection-string")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Spec.ObjectStorageAzureSecret)
		return
	}

	*pulpSettings = *pulpSettings + `AZURE_CONNECTION_STRING = '` + storageData["azure-connection-string"] + `'
AZURE_LOCATION = '` + storageData["azure-container-path"] + `'
AZURE_ACCOUNT_NAME = '` + storageData["azure-account-name"] + `'
AZURE_ACCOUNT_KEY = '` + storageData["azure-account-key"] + `'
AZURE_CONTAINER = '` + storageData["azure-container"] + `'
AZURE_URL_EXPIRATION_SECS = 60
AZURE_OVERWRITE_FILES = True
DEFAULT_FILE_STORAGE = "storages.backends.azure_storage.AzureStorage"
`
}

// s3Settings appends s3 object storage settings into pulpSettings
func s3Settings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	logger := resources.Logger
	context := resources.Context
	client := resources.Client

	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	if storageType[0] != controllers.S3ObjType {
		return
	}

	logger.V(1).Info("Retrieving S3 data from " + resources.Pulp.Spec.ObjectStorageS3Secret)
	storageData, err := controllers.RetrieveSecretData(context, pulp.Spec.ObjectStorageS3Secret, pulp.Namespace, true, client, "s3-access-key-id", "s3-secret-access-key", "s3-bucket-name")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Spec.ObjectStorageS3Secret)
		return
	}

	optionalKey, _ := controllers.RetrieveSecretData(resources.Context, resources.Pulp.Spec.ObjectStorageS3Secret, resources.Pulp.Namespace, false, client, "s3-endpoint", "s3-region")
	if len(optionalKey["s3-endpoint"]) == 0 && len(optionalKey["s3-region"]) == 0 {
		logger.Error(err, "Either s3-endpoint or s3-region needs to be specified", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Spec.ObjectStorageS3Secret)
		return
	}

	if len(optionalKey["s3-endpoint"]) > 0 {
		*pulpSettings = *pulpSettings + fmt.Sprintf("AWS_S3_ENDPOINT_URL = \"%v\"\n", optionalKey["s3-endpoint"])
	}

	if len(optionalKey["s3-region"]) > 0 {
		*pulpSettings = *pulpSettings + fmt.Sprintf("AWS_S3_REGION_NAME = \"%v\"\n", optionalKey["s3-region"])
	}

	*pulpSettings = *pulpSettings + `AWS_ACCESS_KEY_ID = '` + storageData["s3-access-key-id"] + `'
AWS_SECRET_ACCESS_KEY = '` + storageData["s3-secret-access-key"] + `'
AWS_STORAGE_BUCKET_NAME = '` + storageData["s3-bucket-name"] + `'
AWS_DEFAULT_ACL = "@none None"
S3_USE_SIGV4 = True
AWS_S3_SIGNATURE_VERSION = "s3v4"
AWS_S3_ADDRESSING_STYLE = "path"
DEFAULT_FILE_STORAGE = "storages.backends.s3boto3.S3Boto3Storage"
MEDIA_ROOT = ""
`
}

// tokenSettings appends the TOKEN_SERVER setting into pulpSettings
func tokenSettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	rootUrl := getRootURL(resources)

	// configure TOKEN_SERVER based on ingress_type
	tokenServer := "http://" + pulp.Name + "-api-svc." + pulp.Namespace + ".svc.cluster.local:24817/token/"
	if isRoute(pulp) {
		tokenServer = rootUrl + "/token/"
	} else if isIngress(pulp) {
		proto := "http"
		if len(pulp.Spec.IngressTLSSecret) > 0 {
			proto = "https"
		}
		tokenServer = proto + "://" + pulp.Spec.IngressHost + "/token/"
	}
	*pulpSettings = *pulpSettings + fmt.Sprintln("TOKEN_SERVER = \""+tokenServer+"\"")
}

// secretKeySettings appends djange SECRET_KEY setting into pulpSettings
func secretKeySettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	logger := resources.Logger
	pulpSecretKey := pulp.Spec.PulpSecretKey

	logger.V(1).Info("Retrieving Django Secret data from " + pulpSecretKey + " Secret")
	secretKey, err := controllers.RetrieveSecretData(resources.Context, pulpSecretKey, pulp.Namespace, true, resources.Client, "secret_key")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulpSecretKey)
		return
	}

	*pulpSettings = *pulpSettings + fmt.Sprintln("SECRET_KEY = \""+secretKey["secret_key"]+"\"")
}

// allowedContentChecksumsSettings appends the allowed_content_checksums into pulpSettings
func allowedContentChecksumsSettings(resources controllers.FunctionResources, pulpSettings *string) {
	pulp := resources.Pulp
	if len(pulp.Spec.AllowedContentChecksums) == 0 {
		return
	}
	settings, _ := json.Marshal(pulp.Spec.AllowedContentChecksums)
	*pulpSettings = *pulpSettings + fmt.Sprintln("ALLOWED_CONTENT_CHECKSUMS = ", string(settings))
}

// addCustomPulpSettings appends custom settings defined in Pulp CR into pulpSettings
func addCustomPulpSettings(pulp *repomanagerpulpprojectorgv1beta2.Pulp, pulpSettings *string) {
	settings := pulp.Spec.PulpSettings.Raw
	var settingsJson map[string]interface{}
	json.Unmarshal(settings, &settingsJson)

	var convertedSettings string
	sortedKeys := sortKeys(settingsJson)
	for _, k := range sortedKeys {
		if strings.Contains(*pulpSettings, strings.ToUpper(k)) {
			lines := strings.Split(*pulpSettings, strings.ToUpper(k))
			*pulpSettings = lines[0] + strings.Join(strings.Split(lines[1], "\n")[1:], "\n")
		}
		switch settingsJson[k].(type) {
		case map[string]interface{}:
			rawMapping, _ := json.Marshal(settingsJson[k])
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", strings.Replace(string(rawMapping), "\"", "'", -1))
		case []interface{}:
			rawMapping, _ := json.Marshal(settingsJson[k])
			convertedSettings = convertedSettings + fmt.Sprintln(strings.ToUpper(k), "=", string(rawMapping))
		case bool:
			// Pulp expects True or False, but golang boolean values are true or false
			// so we are converting to string and changing to capital T or F
			convertToString := cases.Title(language.English, cases.Compact).String(strconv.FormatBool(settingsJson[k].(bool)))
			convertedSettings = convertedSettings + fmt.Sprintf("%v = %v\n", strings.ToUpper(k), convertToString)
		default:
			// if it is a tuple, we should not parse it as a string (do not add the quotes)
			r, _ := regexp.Compile(`\(.*\)`)
			if r.MatchString(settingsJson[k].(string)) {
				convertedSettings = convertedSettings + fmt.Sprintf("%v = %v\n", strings.ToUpper(k), settingsJson[k])
			} else {
				convertedSettings = convertedSettings + fmt.Sprintf("%v = \"%v\"\n", strings.ToUpper(k), settingsJson[k])
			}
		}
	}

	*pulpSettings = *pulpSettings + convertedSettings
}
