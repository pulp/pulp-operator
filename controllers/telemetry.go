package controllers

import (
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

func verifyTelemetryConfig(resources any, envVars []corev1.EnvVar, containerList []corev1.Container) ([]corev1.EnvVar, []corev1.Container) {
	pulp := resources.(FunctionResources).Pulp

	telemetryImage := os.Getenv("RELATED_OTEL_COLLECTOR_IMAGE")
	if len(pulp.Spec.Telemetry.OpenTelemetryCollectorImage) > 0 && len(pulp.Spec.Telemetry.OpenTelemetryCollectorImageVersion) > 0 {
		telemetryImage = pulp.Spec.Telemetry.OpenTelemetryCollectorImage + ":" + pulp.Spec.Telemetry.OpenTelemetryCollectorImage
	} else if telemetryImage == "" {
		telemetryImage = "docker.io/otel/opentelemetry-collector:latest"
	}

	telemetryEnvVars := []corev1.EnvVar{
		{Name: "PULP_OTEL_ENABLED", Value: strconv.FormatBool(pulp.Spec.Telemetry.Enabled)},
		{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: pulp.Spec.Telemetry.ExporterOtlpProtocol},
		{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: "http://localhost:4318"},
	}

	sidecarTelemetryContainer := corev1.Container{
		Name:            "otel-collector-sidecar",
		Image:           telemetryImage,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Args:            []string{"collector-pulp-api"},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 8889,
			Protocol:      "TCP",
		}},
	}
	// TODO: Create the ConfigMap
	// TODO: Pass the ConfigMap as a volume for otel-collector-config.yml file

	envVars = append(envVars, telemetryEnvVars...)
	containerList = append(containerList, sidecarTelemetryContainer)

	return envVars, containerList
}
