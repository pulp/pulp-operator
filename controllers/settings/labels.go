package settings

import (
	"reflect"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

func PulpcoreLabels(pulp pulpv1.Pulp, pulpcoreType PulpcoreType) map[string]string {
	typeLabel := pulpcoreType.ToLabel()

	labels := map[string]string{
		"app.kubernetes.io/name":       "pulp-" + typeLabel,
		"app.kubernetes.io/instance":   "pulp-" + typeLabel + "-" + pulp.Name,
		"app.kubernetes.io/component":  typeLabel,
		"app.kubernetes.io/part-of":    "pulp",
		"app.kubernetes.io/managed-by": "pulp-operator",
		"app":                          "pulp-" + typeLabel,
		"pulp_cr":                      pulp.Name,
	}

	// Merge custom labels from the CR spec
	customLabels := reflect.ValueOf(pulp.Spec).FieldByName(pulpcoreType.ToField()).FieldByName("CustomLabels").Interface().(map[string]string)
	for k, v := range customLabels {
		labels[k] = v
	}

	return labels
}

func CommonLabels(pulp pulpv1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":    "pulp",
		"app.kubernetes.io/managed-by": "pulp-operator",
		"pulp_cr":                      pulp.Name,
	}
}
