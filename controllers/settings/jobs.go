// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

const (
	migrationJob                = "pulpcore-migration-"
	resetAdminPwdJob            = "reset-admin-password-"
	updateChecksumsJob          = "update-content-checksums-"
	signingScriptJob            = "signing-metadata-"
	ContainerSigningScriptName  = "container_script.sh"
	CollectionSigningScriptName = "collection_script.sh"
)

func MigrationJob(pulpName string) string {
	return pulpName + "-" + migrationJob
}
func ResetAdminPwdJob(pulpName string) string {
	return pulpName + "-" + resetAdminPwdJob
}
func UpdateChecksumsJob(pulpName string) string {
	return pulpName + "-" + updateChecksumsJob
}
func SigningScriptJob(pulpName string) string {
	return pulpName + "-" + signingScriptJob
}
