// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

const (
	adminPassword            = "admin-password"
	djangoSecretKey          = "secret-key"
	containerTokenSecret     = "container-auth"
	pulpServerSecret         = "server"
	dBFieldsEncryptionSecret = "db-fields-encryption"
	rhOperatorPullSecretName = "redhat-operators-pull-secret"
	postgresConfiguration    = "postgres-configuration"
)

func DefaultAdminPassword(pulpName string) string {
	return pulpName + "-" + adminPassword
}
func DefaultDjangoSecretKey(pulpName string) string {
	return pulpName + "-" + djangoSecretKey
}
func DefaultContainerTokenSecret(pulpName string) string {
	return pulpName + "-" + containerTokenSecret
}
func PulpServerSecret(pulpName string) string {
	return pulpName + "-" + pulpServerSecret
}
func DefaultDBFieldsEncryptionSecret(pulpName string) string {
	return pulpName + "-" + dBFieldsEncryptionSecret
}
func RedHatOperatorPullSecret(pulpName string) string {
	return pulpName + "-" + rhOperatorPullSecretName
}
func DefaultDBSecret(pulpName string) string {
	return pulpName + "-" + postgresConfiguration
}

// Default configurations for settings.py
func DefaultPulpSettings(rootUrl string) map[string]string {
	return map[string]string{
		"DB_ENCRYPTION_KEY":         `"/etc/pulp/keys/database_fields.symmetric.key"`,
		"ANSIBLE_CERTS_DIR":         `"/etc/pulp/keys/"`,
		"PRIVATE_KEY_PATH":          `"/etc/pulp/keys/container_auth_private_key.pem"`,
		"PUBLIC_KEY_PATH":           `"/etc/pulp/keys/container_auth_public_key.pem"`,
		"STATIC_ROOT":               `"/var/lib/operator/static/"`,
		"TOKEN_AUTH_DISABLED":       "False",
		"TOKEN_SIGNATURE_ALGORITHM": `"ES256"`,
		"ANSIBLE_API_HOSTNAME":      `"` + rootUrl + `"`,
		"CONTENT_ORIGIN":            `"` + rootUrl + `"`,
	}
}
