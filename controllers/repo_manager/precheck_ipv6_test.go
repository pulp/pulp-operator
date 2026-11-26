package repo_manager

import (
	"testing"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestIpv6DisabledConflictsWithIPFamilies_NoConflictByDefault(t *testing.T) {
	pulp := pulpv1.Pulp{}

	if ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected no conflict when neither field is set")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_NoConflictWhenIPv6NotDisabled(t *testing.T) {
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: []corev1.IPFamily{corev1.IPv6Protocol},
		},
	}

	if ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected no conflict when ipv6_disabled is unset, regardless of ip_families")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_NoConflictWhenFamiliesAreIPv4Only(t *testing.T) {
	disabled := true
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPv6Disabled: &disabled,
			IPFamilies:   []corev1.IPFamily{corev1.IPv4Protocol},
		},
	}

	if ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected no conflict when ipv6_disabled=true and ip_families=[IPv4] agree")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_ConflictWhenIPv6Only(t *testing.T) {
	disabled := true
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPv6Disabled: &disabled,
			IPFamilies:   []corev1.IPFamily{corev1.IPv6Protocol},
		},
	}

	if !ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected conflict when ipv6_disabled=true but ip_families=[IPv6]")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_ConflictWhenDualStackIncludesIPv6(t *testing.T) {
	disabled := true
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPv6Disabled: &disabled,
			IPFamilies:   []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol},
		},
	}

	if !ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected conflict when ipv6_disabled=true but ip_families includes IPv6 (dual-stack)")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_NoConflictWhenExplicitlyFalse(t *testing.T) {
	// Explicitly setting ipv6_disabled=false must behave identically to
	// leaving it unset (nil), not a separate code path.
	disabled := false
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPv6Disabled: &disabled,
			IPFamilies:   []corev1.IPFamily{corev1.IPv6Protocol},
		},
	}

	if ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected no conflict when ipv6_disabled is explicitly false, regardless of ip_families")
	}
}

func TestIpv6DisabledConflictsWithIPFamilies_NoConflictWhenFamiliesExplicitlyEmpty(t *testing.T) {
	// An explicitly empty (non-nil) ip_families slice must default to IPv4,
	// same as unset, not be treated as "no families" with no conflict for
	// the wrong reason.
	disabled := true
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPv6Disabled: &disabled,
			IPFamilies:   []corev1.IPFamily{},
		},
	}

	if ipv6DisabledConflictsWithIPFamilies(pulp) {
		t.Errorf("expected no conflict when ip_families is explicitly empty (defaults to IPv4)")
	}
}

func TestCheckIPFamiliesValid_ValidWhenUnset(t *testing.T) {
	pulp := pulpv1.Pulp{}

	if got := checkIPFamiliesValid(&pulp); got != nil {
		t.Errorf("checkIPFamiliesValid() = %v, want nil (unset is valid)", got)
	}
}

func TestCheckIPFamiliesValid_ValidForIPv4AndIPv6(t *testing.T) {
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: []corev1.IPFamily{corev1.IPv4Protocol, corev1.IPv6Protocol},
		},
	}

	if got := checkIPFamiliesValid(&pulp); got != nil {
		t.Errorf("checkIPFamiliesValid() = %v, want nil", got)
	}
}

func TestCheckIPFamiliesValid_RejectsUnrecognizedValue(t *testing.T) {
	pulp := pulpv1.Pulp{
		Spec: pulpv1.PulpSpec{
			IPFamilies: []corev1.IPFamily{"foo"},
		},
	}

	if got := checkIPFamiliesValid(&pulp); got == nil {
		t.Errorf("checkIPFamiliesValid() = nil, want a non-nil result to block reconciliation on an invalid value")
	}
}
