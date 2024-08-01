package controllers

import (
	"os"
	"reflect"
	"strconv"

	"github.com/pulp/pulp-operator/controllers/settings"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// telemetryConfig adds the otel container sidecar to containers' slice, an otelConfigMap as a new volume, and a pod annotation
func telemetryConfig(resources any, envVars []corev1.EnvVar, containers []corev1.Container, volumes []corev1.Volume, pulpcoreType settings.PulpcoreType) ([]corev1.Container, []corev1.Volume) {
	pulp := resources.(FunctionResources).Pulp
	if !pulp.Spec.Telemetry.Enabled || pulpcoreType != settings.API {
		return containers, volumes
	}

	// set telemetry container image
	telemetryImage := telemetryContainerImage(resources)

	// set telemetry env vars used by pulp-api container
	envVars = append(envVars, telemetryEnvVars(resources)...)

	// update pulp-api container with otel env vars
	containers[0].Env = envVars

	// when telemetry is enabled we need to modify the entrypoint from container image
	containers[0].Command = []string{"/bin/sh", "-c"}
	containers[0].Args = []string{
		`if which pulpcore-api
then
  PULP_API_ENTRYPOINT=("pulpcore-api")
else
  PULP_API_ENTRYPOINT=("gunicorn" "pulpcore.app.wsgi:application" "--bind" "[::]:24817" "--name" "pulp-api" "--access-logformat" "pulp [%({correlation-id}o)s]: %(h)s %(l)s %(u)s %(t)s \"%(r)s\" %(s)s %(b)s \"%(f)s\" \"%(a)s\"")
fi

exec  /usr/local/bin/opentelemetry-instrument --service_name pulp-api "${PULP_API_ENTRYPOINT[@]}" \
--timeout "${PULP_GUNICORN_TIMEOUT}" \
--workers "${PULP_API_WORKERS}" \
--access-logfile -`,
	}

	volumeName := "otel-collector-config"
	// create a volume using the otelconfigmap as source
	telemetryVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: settings.OtelConfigMapName(pulp.Name),
				},
			},
		},
	}

	// set the otel configmap mountpoint
	telemetryVolMount := []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: "/etc/otelcol-contrib/" + settings.OtelConfigFile,
			SubPath:   settings.OtelConfigFile,
			ReadOnly:  true,
		},
	}

	// set resource requirements
	requirements := setResourceRequirements(resources)

	// define the otel container sidecar
	sidecarTelemetryContainer := corev1.Container{
		Name:            "otel-collector-sidecar",
		Image:           telemetryImage,
		ImagePullPolicy: corev1.PullPolicy(pulp.Spec.ImagePullPolicy),
		Ports: []corev1.ContainerPort{{
			ContainerPort: settings.OtelContainerPort,
			Protocol:      "TCP",
		}},
		Args: []string{
			"--config", "file:/etc/otelcol-contrib/" + settings.OtelConfigFile,
		},
		VolumeMounts:    telemetryVolMount,
		Resources:       requirements,
		SecurityContext: SetDefaultSecurityContext(),
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

// setResourceRequirements defines the telemetry container resources
func setResourceRequirements(resources any) corev1.ResourceRequirements {
	pulp := resources.(FunctionResources).Pulp
	requirements := pulp.Spec.Telemetry.ResourceRequirements
	if reflect.DeepEqual(requirements, corev1.ResourceRequirements{}) {
		requirements = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		}
	}
	return requirements
}

// otelConfigMap defines a configmap resource to keep otel-collector-config.yaml configuration file
func OtelConfigMap(resources FunctionResources) client.Object {

	otelConfig := map[string]string{
		settings.OtelConfigFile: `
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
			Name:      settings.OtelConfigMapName(resources.Name),
			Namespace: resources.Namespace,
			Labels:    settings.CommonLabels(*resources.Pulp),
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
	targetPort := intstr.IntOrString{IntVal: settings.OtelContainerPort}
	serviceType := corev1.ServiceType("ClusterIP")
	name := resources.Name
	labels := settings.CommonLabels(*resources.Pulp)
	labels["otel"] = ""

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      settings.OtelServiceName(name),
			Namespace: resources.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			InternalTrafficPolicy: &serviceInternalTrafficPolicyCluster,
			IPFamilies:            []corev1.IPFamily{"IPv4"},
			IPFamilyPolicy:        &ipFamilyPolicyType,
			Ports: []corev1.ServicePort{{
				Name:       "otel-" + strconv.Itoa(settings.OtelContainerPort),
				Port:       settings.OtelContainerPort,
				Protocol:   servicePortProto,
				TargetPort: targetPort,
			}},
			Selector:                 settings.PulpcoreLabels(*resources.Pulp, "api"),
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
	if err := client.Get(ctx, types.NamespacedName{Name: settings.OtelConfigMapName(pulp.Name), Namespace: pulp.Namespace}, otelConfigMap); err == nil {
		client.Delete(ctx, otelConfigMap)
	}

	// remove otel service
	otelService := &corev1.Service{}
	if err := client.Get(ctx, types.NamespacedName{Name: settings.OtelServiceName(pulp.Name), Namespace: pulp.Namespace}, otelService); err == nil {
		client.Delete(ctx, otelService)
	}

	pulp.Status.TelemetryEnabled = false
	client.Status().Update(ctx, pulp)
}
