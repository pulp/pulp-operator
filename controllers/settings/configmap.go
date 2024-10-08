// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

const (
	caConfigMapName = "user-ca-bundle"
)

func EmptyCAConfigMapName(pulpName string) string {
	return pulpName + "-" + caConfigMapName
}

func PulpWebConfigMapName(pulpName string) string {
	return pulpName + "-configmap"
}

func PulpWorkerProbe(pulpName string) string {
	return pulpName + "-worker-probe"
}
