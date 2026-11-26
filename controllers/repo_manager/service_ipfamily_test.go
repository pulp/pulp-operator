package repo_manager

import (
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

// These Services used to hardcode IPFamilies: ["IPv4"] and IPFamilyPolicy:
// SingleStack, which the API server rejects on a single-stack IPv6 cluster
// (pulp/pulp-operator#1380). Leaving both fields unset lets Kubernetes
// default them from the cluster's own Service CIDR instead, the same way
// pulp-web and the redis Service already do.
//
// This can only be verified at the unit level, not via the Ginkgo/envtest
// suite in controller_test.go: envtest's own control plane only has IPv4
// configured, so a live Service fetched after creation resolves to the same
// ["IPv4"]/SingleStack value whether the operator hardcodes it or omits it
// and lets the cluster decide.

func TestServiceAPISpec_DoesNotHardcodeIPFamily(t *testing.T) {
	spec := serviceAPISpec(pulpv1.Pulp{})

	if spec.IPFamilies != nil {
		t.Errorf("serviceAPISpec().IPFamilies = %v, want nil", spec.IPFamilies)
	}
	if spec.IPFamilyPolicy != nil {
		t.Errorf("serviceAPISpec().IPFamilyPolicy = %v, want nil", *spec.IPFamilyPolicy)
	}
}

func TestServiceContentSpec_DoesNotHardcodeIPFamily(t *testing.T) {
	spec := serviceContentSpec(pulpv1.Pulp{})

	if spec.IPFamilies != nil {
		t.Errorf("serviceContentSpec().IPFamilies = %v, want nil", spec.IPFamilies)
	}
	if spec.IPFamilyPolicy != nil {
		t.Errorf("serviceContentSpec().IPFamilyPolicy = %v, want nil", *spec.IPFamilyPolicy)
	}
}

func TestServiceForDatabase_DoesNotHardcodeIPFamily(t *testing.T) {
	svc := serviceForDatabase(&pulpv1.Pulp{})

	if svc.Spec.IPFamilies != nil {
		t.Errorf("serviceForDatabase().Spec.IPFamilies = %v, want nil", svc.Spec.IPFamilies)
	}
	if svc.Spec.IPFamilyPolicy != nil {
		t.Errorf("serviceForDatabase().Spec.IPFamilyPolicy = %v, want nil", *svc.Spec.IPFamilyPolicy)
	}
}
