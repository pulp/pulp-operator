package settings

import (
	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

func PulpcoreLabels(pulp pulpv1.Pulp, pulpType string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "pulp-" + pulpType,
		"app.kubernetes.io/instance":   "pulp-" + pulpType + "-" + pulp.Name,
		"app.kubernetes.io/component":  pulpType,
		"app.kubernetes.io/part-of":    "pulp",
		"app.kubernetes.io/managed-by": "pulp-operator",
		"app":                          "pulp-" + pulpType,
		"pulp_cr":                      pulp.Name,
	}
}

func CommonLabels(pulp pulpv1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":    "pulp",
		"app.kubernetes.io/managed-by": "pulp-operator",
		"pulp_cr":                      pulp.Name,
	}
}
