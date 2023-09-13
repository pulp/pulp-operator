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
