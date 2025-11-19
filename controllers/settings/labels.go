package settings

import (
	"maps"
	"reflect"

	pulpv1 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1"
)

func PulpcorePodLabels(pulp pulpv1.Pulp, pulpcoreType PulpcoreType) map[string]string {
	podLabels := reflect.ValueOf(pulp.Spec).FieldByName(pulpcoreType.ToField()).FieldByName("PodLabels").Interface().(map[string]string)
	if podLabels == nil {
		podLabels = make(map[string]string)
	}

	// Merge with PulpcorePodLabels, PulpcoreLabels has precedence
	maps.Copy(podLabels, PulpcoreLabels(pulp, pulpcoreType))

	return podLabels
}

func PulpcoreLabels(pulp pulpv1.Pulp, pulpcoreType PulpcoreType) map[string]string {
	typeLabel := pulpcoreType.ToLabel()

	labels := map[string]string{
		"app.kubernetes.io/name":      "pulp-" + typeLabel,
		"app.kubernetes.io/instance":  "pulp-" + typeLabel + "-" + pulp.Name,
		"app.kubernetes.io/component": typeLabel,
		"app":                         "pulp-" + typeLabel,
	}

	// Merge with CommonLabels, CommonLabels has precedence
	maps.Copy(labels, CommonLabels(pulp))

	return labels
}

func CommonLabels(pulp pulpv1.Pulp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/part-of":    "pulp",
		"app.kubernetes.io/managed-by": "pulp-operator",
		"pulp_cr":                      pulp.Name,
	}
}
