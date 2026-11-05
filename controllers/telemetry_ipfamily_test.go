package controllers

import (
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// ServiceOtel used to hardcode IPFamilies: ["IPv4"] and IPFamilyPolicy:
// SingleStack, which the API server rejects on a single-stack IPv6 cluster
// (pulp/pulp-operator#1380). Leaving both fields unset lets Kubernetes
// default them from the cluster's own Service CIDR instead, the same way
// pulp-web and the redis Service already do.
//
// This can only be verified at the unit level, not via an envtest-based
// suite: envtest's own control plane only has IPv4 configured, so a live
// Service fetched after creation resolves to the same ["IPv4"]/SingleStack
// value whether the operator hardcodes it or omits it and lets the cluster
// decide.
func TestServiceOtel_DoesNotHardcodeIPFamily(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = pulpv1.AddToScheme(scheme)

	pulp := &pulpv1.Pulp{ObjectMeta: metav1.ObjectMeta{Name: "pulp", Namespace: "default"}}
	resources := FunctionResources{Pulp: pulp, Scheme: scheme}

	obj := ServiceOtel(resources)
	svc := obj.(*corev1.Service)

	if svc.Spec.IPFamilies != nil {
		t.Errorf("ServiceOtel().Spec.IPFamilies = %v, want nil", svc.Spec.IPFamilies)
	}
	if svc.Spec.IPFamilyPolicy != nil {
		t.Errorf("ServiceOtel().Spec.IPFamilyPolicy = %v, want nil", *svc.Spec.IPFamilyPolicy)
	}
}
