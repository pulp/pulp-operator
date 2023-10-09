package settings

import (
	repomanagerpulpprojectorgv1beta2 "github.com/pulp/pulp-operator/apis/repo-manager.pulpproject.org/v1beta2"
)

func PulpcoreLabels(pulp repomanagerpulpprojectorgv1beta2.Pulp, pulpType string) map[string]string {
	deploymentType := pulp.Spec.DeploymentType
	return map[string]string{
		"app.kubernetes.io/name":       deploymentType + "-" + pulpType,
		"app.kubernetes.io/instance":   deploymentType + "-" + pulpType + "-" + pulp.Name,
		"app.kubernetes.io/component":  pulpType,
		"app.kubernetes.io/part-of":    deploymentType,
		"app.kubernetes.io/managed-by": deploymentType + "-operator",
		"app":                          "pulp-" + pulpType,
		"pulp_cr":                      pulp.Name,
	}
}

func CommonLabels(pulp repomanagerpulpprojectorgv1beta2.Pulp) map[string]string {
	deploymentType := pulp.Spec.DeploymentType
	return map[string]string{
		"app.kubernetes.io/part-of":    deploymentType,
		"app.kubernetes.io/managed-by": deploymentType + "-operator",
		"pulp_cr":                      pulp.Name,
	}
}
