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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	"github.com/pulp/pulp-operator/controllers"
	"github.com/pulp/pulp-operator/controllers/settings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *RepoManagerReconciler) createSecrets(ctx context.Context, pulp *pulpv1.Pulp) (*ctrl.Result, error) {

	// conditionType is used to update .status.conditions with the current resource state
	conditionType := "Pulp-API-Ready"

	// if .spec.admin_password_secret is not defined, operator will default to pulp-admin-password
	adminSecretName := settings.DefaultAdminPassword(pulp.Name)
	if len(pulp.Spec.AdminPasswordSecret) > 1 {
		adminSecretName = pulp.Spec.AdminPasswordSecret
	}
	// update pulp CR admin-password secret with default name
	if err := controllers.UpdateCRField(ctx, r.Client, pulp, "AdminPasswordSecret", adminSecretName); err != nil {
		return &ctrl.Result{}, err
	}

	// if .spec.pulp_secret_key is not defined, operator will default to "pulp-secret-key"
	djangoKey := settings.DefaultDjangoSecretKey(pulp.Name)
	if len(pulp.Spec.PulpSecretKey) > 0 {
		djangoKey = pulp.Spec.PulpSecretKey
	}
	// update pulp CR pulp_secret_key secret with default name
	if err := controllers.UpdateCRField(ctx, r.Client, pulp, "PulpSecretKey", djangoKey); err != nil {
		return &ctrl.Result{}, err
	}

	// update pulp CR with default values
	dbFieldsEncryptionSecret := settings.DefaultDBFieldsEncryptionSecret(pulp.Name)
	if len(pulp.Spec.DBFieldsEncryptionSecret) > 0 {
		dbFieldsEncryptionSecret = pulp.Spec.DBFieldsEncryptionSecret
	}
	if err := controllers.UpdateCRField(ctx, r.Client, pulp, "DBFieldsEncryptionSecret", dbFieldsEncryptionSecret); err != nil {
		return &ctrl.Result{}, err
	}

	// update pulp CR with container_token_secret secret value
	containerTokenSecret := settings.DefaultContainerTokenSecret(pulp.Name)
	if len(pulp.Spec.ContainerTokenSecret) > 0 {
		containerTokenSecret = pulp.Spec.ContainerTokenSecret
	}
	if err := controllers.UpdateCRField(ctx, r.Client, pulp, "ContainerTokenSecret", containerTokenSecret); err != nil {
		return &ctrl.Result{}, err
	}

	serverSecretName := settings.PulpServerSecret(pulp.Name)

	// list of pulp-api resources that should be provisioned
	resources := []ApiResource{
		// pulp-secret-key secret
		{ResourceDefinition{ctx, &corev1.Secret{}, djangoKey, "PulpSecretKey", conditionType, pulp}, pulpDjangoKeySecret},
		// pulp-server secret
		{Definition: ResourceDefinition{Context: ctx, Type: &corev1.Secret{}, Name: serverSecretName, Alias: "Server", ConditionType: conditionType, Pulp: pulp}, Function: pulpServerSecret},
		// pulp-db-fields-encryption secret
		{ResourceDefinition{ctx, &corev1.Secret{}, dbFieldsEncryptionSecret, "DBFieldsEncryptionSecret", conditionType, pulp}, pulpDBFieldsEncryptionSecret},
		// pulp-admin-password secret
		{ResourceDefinition{ctx, &corev1.Secret{}, adminSecretName, "AdminPassword", conditionType, pulp}, pulpAdminPasswordSecret},
		// pulp-container-auth secret
		{ResourceDefinition{ctx, &corev1.Secret{}, containerTokenSecret, "ContainerTokenSecret", conditionType, pulp}, pulpContainerAuth},
	}

	// create the secrets
	for _, resource := range resources {
		requeue, err := r.createPulpResource(resource.Definition, resource.Function)
		if err != nil {
			return &ctrl.Result{}, err
		} else if requeue {
			return &ctrl.Result{Requeue: true}, nil
		}
	}

	// Ensure the secret data is as expected
	funcResources := controllers.FunctionResources{Context: ctx, Client: r.Client, Pulp: pulp, Scheme: r.Scheme, Logger: r.RawLogger}
	serverSecret := &corev1.Secret{}
	r.Get(ctx, types.NamespacedName{Name: serverSecretName, Namespace: pulp.Namespace}, serverSecret)
	expectedServerSecret := pulpServerSecret(funcResources)
	if requeue, err := controllers.ReconcileObject(funcResources, expectedServerSecret, serverSecret, conditionType, controllers.PulpSecret{}); err != nil || requeue {
		// restart pulpcore pods if the secret has changed
		r.restartPulpCorePods(ctx, pulp)
		return &ctrl.Result{Requeue: requeue}, err
	}

	return nil, nil
}

// pulpServerSecret creates the pulp-server secret object which is used to
// populate the /etc/pulp/settings.py config file
func pulpServerSecret(resources controllers.FunctionResources) client.Object {

	pulp := resources.Pulp
	pulp_settings := controllers.DotNotEditMessage

	// add custom settings to the secret
	customSettings := addCustomPulpSettings(resources, &pulp_settings)

	// pulpcore debug log
	debugLogging(resources, &pulp_settings)

	// db settings
	databaseSettings(resources, &pulp_settings, customSettings)

	// add cache settings
	cacheSettings(resources, &pulp_settings)

	// azure settings
	azureSettings(resources, &pulp_settings, customSettings)

	// gcs settings
	gcsSettings(resources, &pulp_settings, customSettings)

	// s3 settings
	s3Settings(resources, &pulp_settings, customSettings)

	// configure settings.py with keycloak integration variables
	ssoConfig(resources, &pulp_settings)

	// configure TOKEN_SERVER based on ingress_type
	tokenSettings(resources, &pulp_settings, customSettings)

	// django SECRET_KEY
	secretKeySettings(resources, &pulp_settings, customSettings)

	// allowed content checksum
	allowedContentChecksumsSettings(resources, &pulp_settings, customSettings)

	// ldap auth config
	ldapSettings(resources, &pulp_settings)

	// pulp configurations that need to run a django migrations
	needsMigrationSetting(resources, &pulp_settings, customSettings)

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
func databaseSettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["DATABASES"]; exists {
		return
	}

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
func azureSettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["STORAGES"]; exists {
		return
	}

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

	*pulpSettings = *pulpSettings + `MEDIA_ROOT = ""
STORAGES = {
    "default": {
        "BACKEND": "storages.backends.azure_storage.AzureStorage",
        "OPTIONS": {
            "connection_string": '` + storageData["azure-connection-string"] + `',
            "account_name": '` + storageData["azure-account-name"] + `',
            "azure_container": '` + storageData["azure-container"] + `',
            "account_key": '` + storageData["azure-account-key"] + `',
            "expiration_secs": 60,
            "overwrite_files": 'True',
            "location": '` + storageData["azure-container-path"] + `'
        },
    },
    "staticfiles": {"BACKEND": "django.contrib.staticfiles.storage.StaticFilesStorage"},
}
`

}

// gcsSettings appends GCS object storage settings into pulpSettings
func gcsSettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["STORAGES"]; exists {
		return
	}

	pulp := resources.Pulp
	logger := resources.Logger
	context := resources.Context
	client := resources.Client

	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	if storageType[0] != controllers.GCSObjType {
		return
	}

	logger.V(1).Info("Retrieving GCS data from " + resources.Pulp.Spec.ObjectStorageGCSSecret)

	// Bucket name is a mandatory field
	storageData, err := controllers.RetrieveSecretData(context, pulp.Spec.ObjectStorageGCSSecret, pulp.Namespace, true, client, "gcs-bucket-name")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Spec.ObjectStorageGCSSecret)
		return
	}

	// Add optional keys
	var gcsIamSignBlob, gcsQueryStringAuth, gcsFileOverwrite string
	optionalKey, _ := controllers.RetrieveSecretData(context, pulp.Spec.ObjectStorageGCSSecret, pulp.Namespace, false, client, "gcs-iam-sign-blob", "gcs-querystring-auth", "gcs-file-overwrite")
	if len(optionalKey["gcs-iam-sign-blob"]) > 0 {
		gcsIamSignBlob = fmt.Sprintf("%12s\"iam_sign_blob\": %v,\n", "", optionalKey["gcs-iam-sign-blob"])
	}
	if len(optionalKey["gcs-querystring-auth"]) > 0 {
		gcsQueryStringAuth = fmt.Sprintf("%12s\"querystring_auth\": %v,\n", "", optionalKey["gcs-querystring-auth"])
	}

	if len(optionalKey["gcs-file-overwrite"]) > 0 {
		gcsFileOverwrite = fmt.Sprintf("%12s\"file_overwrite\": %v,\n", "", optionalKey["gcs-file-overwrite"])
	}

	gcsOptions := `        "OPTIONS": {
            "bucket_name": '` + storageData["gcs-bucket-name"] + `',
`

	gcsOptions += gcsIamSignBlob
	gcsOptions += gcsQueryStringAuth
	gcsOptions += gcsFileOverwrite
	gcsOptions += fmt.Sprintf("%8s},\n", "")

	*pulpSettings = *pulpSettings + `MEDIA_ROOT = ""
STORAGES = {
    "default": {
        "BACKEND": "storages.backends.gcloud.GoogleCloudStorage",
`
	*pulpSettings += gcsOptions
	*pulpSettings += `    },
    "staticfiles": {"BACKEND": "django.contrib.staticfiles.storage.StaticFilesStorage"},
`
	*pulpSettings += fmt.Sprintln("}")
}

// s3Settings appends s3 object storage settings into pulpSettings
func s3Settings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["STORAGES"]; exists {
		return
	}
	pulp := resources.Pulp
	logger := resources.Logger
	context := resources.Context
	client := resources.Client

	_, storageType := controllers.MultiStorageConfigured(pulp, "Pulp")
	if storageType[0] != controllers.S3ObjType {
		return
	}

	logger.V(1).Info("Retrieving S3 data from " + resources.Pulp.Spec.ObjectStorageS3Secret)
	storageData, err := controllers.RetrieveSecretData(context, pulp.Spec.ObjectStorageS3Secret, pulp.Namespace, true, client, "s3-bucket-name")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulp.Spec.ObjectStorageS3Secret)
		return
	}

	optionalKey, _ := controllers.RetrieveSecretData(resources.Context, resources.Pulp.Spec.ObjectStorageS3Secret, resources.Pulp.Namespace, false, client, "s3-endpoint", "s3-region", "s3-access-key-id", "s3-secret-access-key", "s3-addressing-style")
	if len(optionalKey["s3-endpoint"]) == 0 && len(optionalKey["s3-region"]) == 0 {
		logger.Error(err, "Either s3-endpoint or s3-region needs to be specified", "Secret.Namespace", resources.Pulp.Namespace, "Secret.Name", resources.Pulp.Spec.ObjectStorageS3Secret)
		return
	}

	var s3SecretKey, s3KeyId, s3Endpoint, s3Region string
	if len(optionalKey["s3-secret-access-key"]) > 0 {
		s3SecretKey = fmt.Sprintf("%12s\"secret_key\": \"%v\",\n", "", optionalKey["s3-secret-access-key"])
	}

	if len(optionalKey["s3-access-key-id"]) > 0 {
		s3KeyId = fmt.Sprintf("%12s\"access_key\": \"%v\",\n", "", optionalKey["s3-access-key-id"])
	}

	if len(optionalKey["s3-endpoint"]) > 0 {
		s3Endpoint = fmt.Sprintf("%12s\"endpoint_url\": \"%v\",\n", "", optionalKey["s3-endpoint"])
	}

	if len(optionalKey["s3-region"]) > 0 {
		s3Region = fmt.Sprintf("%12s\"region_name\": \"%v\",\n", "", optionalKey["s3-region"])
	}

	s3Options := `        "OPTIONS": {
            "signature_version": "s3v4",
            "addressing_style": "` + getAddressingStyle(optionalKey) + `",
            "bucket_name": '` + storageData["s3-bucket-name"] + `',
`
	s3Options += s3SecretKey
	s3Options += s3KeyId
	s3Options += s3Endpoint
	s3Options += s3Region
	s3Options += fmt.Sprintf("%8s},\n", "")

	*pulpSettings += `MEDIA_ROOT = ""
STORAGES = {
    "default": {
        "BACKEND": "storages.backends.s3boto3.S3Boto3Storage",
`
	*pulpSettings += s3Options
	*pulpSettings += `    },
    "staticfiles": {"BACKEND": "django.contrib.staticfiles.storage.StaticFilesStorage"},
`
	*pulpSettings += fmt.Sprintln("}")

}

// tokenSettings appends the TOKEN_SERVER setting into pulpSettings
func tokenSettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["TOKEN_SERVER"]; exists {
		return
	}

	pulp := resources.Pulp
	rootUrl := getRootURL(*pulp)

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
func secretKeySettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["SECRET_KEY"]; exists {
		return
	}

	pulp := resources.Pulp
	logger := resources.Logger
	pulpSecretKey := pulp.Spec.PulpSecretKey

	logger.V(1).Info("Retrieving Django Secret data from " + pulpSecretKey + " Secret")
	secretKey, err := controllers.RetrieveSecretData(resources.Context, pulpSecretKey, pulp.Namespace, true, resources.Client, "secret_key")
	if err != nil {
		logger.Error(err, "Secret Not Found!", "Secret.Namespace", pulp.Namespace, "Secret.Name", pulpSecretKey)
		return
	}

	*pulpSettings = *pulpSettings + fmt.Sprintf("SECRET_KEY = \"%v\"\n", secretKey["secret_key"])
}

// allowedContentChecksumsSettings appends the allowed_content_checksums into pulpSettings
func allowedContentChecksumsSettings(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	if _, exists := customSettings["ALLOWED_CONTENT_CHECKSUMS"]; exists {
		return
	}

	pulp := resources.Pulp
	if len(pulp.Spec.AllowedContentChecksums) == 0 {
		return
	}
	settings, _ := json.Marshal(pulp.Spec.AllowedContentChecksums)
	*pulpSettings = *pulpSettings + fmt.Sprintln("ALLOWED_CONTENT_CHECKSUMS = ", string(settings))
}

// addCustomPulpSettings defines settings.py with the configurations defined in custom_pulp_settings configmap
// and returns a map with all the custom keys defined
func addCustomPulpSettings(resources controllers.FunctionResources, pulpSettings *string) map[string]struct{} {
	pulp := resources.Pulp
	rootUrl := getRootURL(*pulp)
	defaultSettings := settings.DefaultPulpSettings(rootUrl)

	// if custom_pulp_settings is not defined, append the default values and return
	if pulp.Spec.CustomPulpSettings == "" {
		for _, k := range sortKeys(defaultSettings) {
			*pulpSettings = *pulpSettings + fmt.Sprintf("%v = %v\n", k, defaultSettings[k])
		}
		return nil
	}

	settingsCM := &corev1.ConfigMap{}
	resources.Client.Get(resources.Context, types.NamespacedName{Name: pulp.Spec.CustomPulpSettings, Namespace: pulp.Namespace}, settingsCM)

	// store the keys found in custom_pulp_settings configmap
	settings := map[string]struct{}{}
	for _, k := range sortKeys(settingsCM.Data) {
		configName, configValue := strings.ToUpper(k), settingsCM.Data[k]
		*pulpSettings = *pulpSettings + fmt.Sprintf("%v = %v\n", configName, configValue)
		settings[strings.ToUpper(k)] = struct{}{}

		// remove the settings from defaultSettings dict to avoid duplicate config
		delete(defaultSettings, strings.ToUpper(k))
	}

	for _, k := range sortKeys(defaultSettings) {
		*pulpSettings = *pulpSettings + fmt.Sprintf("%v = %v\n", k, defaultSettings[k])
	}
	return settings
}

// debugLogging will set the log level from Pulpcore pods to DEBUG
func debugLogging(resources controllers.FunctionResources, pulpSettings *string) {

	if resources.Pulp.Spec.EnableDebugging {
		*pulpSettings = *pulpSettings + fmt.Sprintln("LOGGING = {'dynaconf_merge': True, 'loggers': {'': {'handlers': ['console'], 'level': 'DEBUG'}}}")
	}
}

// needsMigrationSetting defines settings.py with some specific configurations that when changed will
// also trigger a migration job
func needsMigrationSetting(resources controllers.FunctionResources, pulpSettings *string, customSettings map[string]struct{}) {
	for operatorFieldName, pulpFieldName := range controllers.MigrationSettingsList() {
		customSettingsFound := false
		if _, exists := customSettings[pulpFieldName]; exists {
			logMessage := fmt.Sprintf("%v should not be defined in custom_pulp_settings. Use pulp.Spec.%v instead", pulpFieldName, strings.ToLower(pulpFieldName))
			controllers.CustomZapLogger().Warn(logMessage)
			customSettingsFound = true
		}
		if customSettingsFound {
			continue
		}

		config := reflect.ValueOf(resources.Pulp.Spec).FieldByName(operatorFieldName).Bool()
		if !config {
			return
		}
		configCapitalized := cases.Title(language.English, cases.Compact).String(strconv.FormatBool(config))
		*pulpSettings = *pulpSettings + fmt.Sprintf("%v = %v\n", pulpFieldName, configCapitalized)

	}
}

// getAddressingStyle returns the addressing style from secret or defaults to "path"
func getAddressingStyle(optionalKey map[string]string) string {
	if addressingStyle, exists := optionalKey["s3-addressing-style"]; exists && len(addressingStyle) > 0 {
		return addressingStyle
	}
	return "path" // default value
}
