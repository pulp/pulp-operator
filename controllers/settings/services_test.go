package settings

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

func TestIPFamilyPolicyDefaultsToSingleStackWhenUnset(t *testing.T) {
	pulp := pulpv1.Pulp{}

	got := IPFamilyPolicy(pulp)

	if got == nil {
		t.Fatalf("IPFamilyPolicy() = nil, want a pointer to SingleStack")
	}
	if *got != corev1.IPFamilyPolicySingleStack {
		t.Errorf("IPFamilyPolicy() = %q, want %q", *got, corev1.IPFamilyPolicySingleStack)
	}
}

func TestIPFamilyPolicyPassesThroughCROverride(t *testing.T) {
	want := corev1.IPFamilyPolicyPreferDualStack
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilyPolicy: &want,
		},
	}

	got := IPFamilyPolicy(pulp)

	if got == nil {
		t.Fatalf("IPFamilyPolicy() = nil, want %q", want)
	}
	if *got != want {
		t.Errorf("IPFamilyPolicy() = %q, want %q", *got, want)
	}
}

func TestIPFamilyPolicyDoesNotAliasCRField(t *testing.T) {
	// The returned pointer must be a copy: mutating it must not mutate the
	// Pulp CR's own spec field.
	original := corev1.IPFamilyPolicyPreferDualStack
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilyPolicy: &original,
		},
	}

	got := IPFamilyPolicy(pulp)
	*got = corev1.IPFamilyPolicyRequireDualStack

	if *pulp.Spec.IPFamilyPolicy != corev1.IPFamilyPolicyPreferDualStack {
		t.Errorf("mutating the returned pointer changed pulp.Spec.IPFamilyPolicy to %q, want it unchanged at %q",
			*pulp.Spec.IPFamilyPolicy, corev1.IPFamilyPolicyPreferDualStack)
	}
}

func TestIPFamiliesDefaultsToIPv4OnlyWhenUnset(t *testing.T) {
	pulp := pulpv1.Pulp{}

	got := IPFamilies(pulp)

	want := []corev1.IPFamily{corev1.IPv4Protocol}
	if !slices.Equal(got, want) {
		t.Errorf("IPFamilies() = %v, want %v", got, want)
	}
}

func TestIPFamiliesDefaultsToIPv4OnlyWhenExplicitlyEmptySlice(t *testing.T) {
	// A non-nil but empty slice should be treated the same as unset.
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: []corev1.IPFamily{},
		},
	}

	got := IPFamilies(pulp)

	want := []corev1.IPFamily{corev1.IPv4Protocol}
	if !slices.Equal(got, want) {
		t.Errorf("IPFamilies() = %v, want %v", got, want)
	}
}

func TestIPFamiliesPassesThroughCROverride(t *testing.T) {
	want := []corev1.IPFamily{corev1.IPv6Protocol}
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: want,
		},
	}

	got := IPFamilies(pulp)

	if !slices.Equal(got, want) {
		t.Errorf("IPFamilies() = %v, want %v", got, want)
	}
}

func TestIPFamiliesPassesThroughDualStackOrder(t *testing.T) {
	// Order matters: it determines the primary family under
	// PreferDualStack/RequireDualStack. Confirm it's passed through
	// unmodified, not reordered or deduplicated.
	want := []corev1.IPFamily{corev1.IPv6Protocol, corev1.IPv4Protocol}
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: want,
		},
	}

	got := IPFamilies(pulp)

	if !slices.Equal(got, want) {
		t.Errorf("IPFamilies() = %v, want %v", got, want)
	}
}

func TestIPFamiliesDoesNotAliasCRField(t *testing.T) {
	// The returned slice must be a copy: mutating it must not mutate the
	// Pulp CR's own spec field.
	original := []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: original,
		},
	}

	got := IPFamilies(pulp)
	got[0] = corev1.IPv6Protocol

	want := []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol}
	if !slices.Equal(pulp.Spec.IPFamilies, want) {
		t.Errorf("mutating the returned slice changed pulp.Spec.IPFamilies to %v, want it unchanged at %v",
			pulp.Spec.IPFamilies, want)
	}
}
