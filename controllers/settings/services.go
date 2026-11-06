// This file contains resource names and constants that are used to provision
// the Kubernetes objects. We are centralizing them here to make it easier to
// maintain and, in case we decide to support multiple CRs running in the same
// namespace, to avoid name colision or code repetition.
// Since go const does not allow to pass variables and there is no immutable vars
// we are encapsulating the constants in each function to return a value based
// on Pulp CR name.

package settings

import (
	corev1 "k8s.io/api/core/v1"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

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

// IPFamilyPolicy resolves the IP family policy to use for a Service from the
// Pulp CR spec, defaulting to SingleStack (current, pre-existing behavior)
// when the field is unset. Returns a fresh pointer rather than aliasing
// pulp.Spec.IPFamilyPolicy, so callers can't accidentally mutate the CR spec
// through it.
func IPFamilyPolicy(pulp pulpv1.Pulp) *corev1.IPFamilyPolicy {
	if pulp.Spec.IPFamilyPolicy != nil {
		policy := *pulp.Spec.IPFamilyPolicy
		return &policy
	}
	policy := corev1.IPFamilyPolicySingleStack
	return &policy
}

// IPFamilies resolves the IP families to assign to a Service from the Pulp CR
// spec, defaulting to ["IPv4"] (current, pre-existing behavior) when the
// field is unset. The operator does not validate this against what the
// cluster's CNI actually supports; it is passed straight through. Returns a
// copy rather than aliasing pulp.Spec.IPFamilies, so callers can't
// accidentally mutate the CR spec through the returned slice.
func IPFamilies(pulp pulpv1.Pulp) []corev1.IPFamily {
	if len(pulp.Spec.IPFamilies) > 0 {
		families := make([]corev1.IPFamily, len(pulp.Spec.IPFamilies))
		copy(families, pulp.Spec.IPFamilies)
		return families
	}
	return []corev1.IPFamily{corev1.IPv4Protocol}
}
