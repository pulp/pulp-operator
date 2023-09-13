// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

func ApiService(pulpName string) string {
	return pulpName + "-api-svc"
}
func ContentService(pulpName string) string {
	return pulpName + "-content-svc"
}
func WorkerService(pulpName string) string {
	return pulpName + "-worker-svc"
}
func PulpWebService(pulpName string) string {
	return pulpName + "-web-svc"
}
func DBService(pulpName string) string {
	return pulpName + "-database-svc"
}
func CacheService(pulpName string) string {
	return pulpName + "-redis-svc"
}
