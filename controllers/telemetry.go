package controllers

import (
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const OtelConfigName = "otel-collector-config"
const OtelConfigFile = "otel-collector-config.yaml"
const OtelServiceName = "otel-collector-svc"
const otelContainerPort = 8889

// telemetryConfig adds the otel container sidecar to containers' slice, an otelConfigMap as a new volume, and a pod annotation
func telemetryConfig(resources any, envVars []corev1.EnvVar, containers []corev1.Container, volumes []corev1.Volume) ([]corev1.Container, []corev1.Volume) {
	pulp := resources.(FunctionResources).Pulp

	// set telemetry container image
	telemetryImage := telemetryContainerImage(resources)

	// set telemetry env vars used by pulp-api container
	envVars = append(envVars, telemetryEnvVars(resources)...)

	// update pulp-api container with otel env vars
	containers[0].Env = envVars

	// when telemetry is enabled we need to modify the entrypoint from container image
	containers[0].Command = []string{"/bin/sh", "-c"}
	containers[0].Args = []string{
		`exec /usr/local/bin/opentelemetry-instrument --service_name pulp-api gunicorn --bind '[::]:24817' pulpcore.app.wsgi:application --name pulp-api --timeout "${PULP_GUNICORN_TIMEOUT}" --workers "${PULP_API_WORKERS}"`,
	}

	// create a volume using the otelconfigmap as source
	telemetryVolume := corev1.Volume{
		Name: OtelConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: OtelConfigName,
				},
			},
		},
	}

	// set the otel configmap mountpoint
	telemetryVolMount := []corev1.VolumeMount{
		{
			Name:      OtelConfigName,
			MountPath: "/etc/otelcol-contrib/" + OtelConfigFile,
			SubPath:   OtelConfigFile,
			ReadOnly:  true,
		},
	}

	// define the otel container sidecar
	sidecarTelemetryContainer := corev1.Container{
		Name:            "otel-collector-sidecar",
		Image:           telemetryImage,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Ports: []corev1.ContainerPort{{
			ContainerPort: otelContainerPort,
			Protocol:      "TCP",
		}},
		Args: []string{
			"--config", "file:/etc/otelcol-contrib/" + OtelConfigFile,
		},
		VolumeMounts: telemetryVolMount,
	}

	containers = append(containers, sidecarTelemetryContainer)
	volumes = append(volumes, telemetryVolume)

	return containers, volumes
}

// telemetryContainerImage defines the image that will be used by otel_collector sidecar container
func telemetryContainerImage(resources any) string {
	pulp := resources.(FunctionResources).Pulp

	// set telemetry container image
	telemetryImage := os.Getenv("RELATED_OTEL_COLLECTOR_IMAGE")
	if len(pulp.Spec.Telemetry.OpenTelemetryCollectorImage) > 0 && len(pulp.Spec.Telemetry.OpenTelemetryCollectorImageVersion) > 0 {
		return pulp.Spec.Telemetry.OpenTelemetryCollectorImage + ":" + pulp.Spec.Telemetry.OpenTelemetryCollectorImageVersion
	} else if telemetryImage == "" {
		return "docker.io/otel/opentelemetry-collector:latest"
	}

	return telemetryImage
}

// telemetryEnvVars defines the env vars that needs to be added to pulp containers to export the metrics
func telemetryEnvVars(resources any) []corev1.EnvVar {
	pulp := resources.(FunctionResources).Pulp
	return []corev1.EnvVar{
		{Name: "PULP_OTEL_ENABLED", Value: strconv.FormatBool(pulp.Spec.Telemetry.Enabled)},
		{Name: "OTEL_EXPORTER_OTLP_PROTOCOL", Value: pulp.Spec.Telemetry.ExporterOtlpProtocol},
		{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: "http://localhost:4318"},
	}
}

// otelConfigMap defines a configmap resource to keep otel-collector-config.yaml configuration file
func OtelConfigMap(resources FunctionResources) client.Object {

	otelConfig := map[string]string{
		OtelConfigFile: `
receivers:
  otlp:
    protocols:
      http:

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  otlp:
    endpoint: localhost:4317

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: []
      exporters: [prometheus]
    traces:
      receivers: [otlp]
      processors: []
      exporters: [otlp]
`,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OtelConfigName,
			Namespace: resources.Namespace,
		},
		Data: otelConfig,
	}
	ctrl.SetControllerReference(resources.Pulp, cm, resources.Scheme)
	return cm
}

// serviceOtel defines a service to expose otel metrics
func ServiceOtel(resources FunctionResources) client.Object {
	serviceInternalTrafficPolicyCluster := corev1.ServiceInternalTrafficPolicyType("Cluster")
	ipFamilyPolicyType := corev1.IPFamilyPolicyType("SingleStack")
	serviceAffinity := corev1.ServiceAffinity("None")
	servicePortProto := corev1.Protocol("TCP")
	targetPort := intstr.IntOrString{IntVal: otelContainerPort}
	serviceType := corev1.ServiceType("ClusterIP")
	deployment_type := resources.Pulp.Spec.DeploymentType
	name := resources.Name

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      OtelServiceName,
			Namespace: resources.Namespace,
			Labels: map[string]string{
				"otel": "",
			},
		},
		Spec: corev1.ServiceSpec{
			InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
			IPFamilies:            []corev1.IPFamily{"IPv4"},
			IPFamilyPolicy:        &ipFamilyPolicyType,
			Ports: []corev1.ServicePort{{
				Name:       "otel-" + strconv.Itoa(otelContainerPort),
				Port:       otelContainerPort,
				Protocol:   servicePortProto,
				TargetPort: targetPort,
			}},
			Selector: map[string]string{
				"app.kubernetes.io/name":       deployment_type + "-api",
				"app.kubernetes.io/instance":   deployment_type + "-api-" + name,
				"app.kubernetes.io/component":  "api",
				"app.kubernetes.io/part-of":    deployment_type,
				"app.kubernetes.io/managed-by": deployment_type + "-operator",
				"app":                          "pulp-api",
				"pulp_cr":                      name,
			},
			SessionAffinity:          serviceAffinity,
			Type:                     serviceType,
			PublishNotReadyAddresses: true,
		},
	}

	ctrl.SetControllerReference(resources.Pulp, svc, resources.Scheme)
	return svc
}

// RemoveTelemetryResources cleans up telemetry resources if telemetry.enabled == false
func RemoveTelemetryResources(resources FunctionResources) {
	client := resources.Client
	ctx := resources.Context
	pulp := resources.Pulp

	// remove otel configmap
	otelConfigMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, types.NamespacedName{Name: OtelConfigName, Namespace: pulp.Namespace}, otelConfigMap); err == nil {
		client.Delete(ctx, otelConfigMap)
	}

	// remove otel service
	otelService := &corev1.Service{}
	if err := client.Get(ctx, types.NamespacedName{Name: OtelServiceName, Namespace: pulp.Namespace}, otelService); err == nil {
		client.Delete(ctx, otelService)
	}

	pulp.Status.TelemetryEnabled = false
	client.Status().Update(ctx, pulp)
}
