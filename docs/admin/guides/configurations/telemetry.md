# Enable metrics collection

!!! info
    The following steps were tested **only in OpenShift** environments.  
    For other kubernetes distributions (*eks*, *minikube*, *kind* ,etc) the instructions
    can be different and were not tested by Pulp team yet.

Pulp operator allows to enable Pulp metrics collection using the [OpenTelemetry Framework](https://opentelemetry.io/) and exposing them to Prometheus.

### *Prerequisites*

* the [monitoring for user-defined projects](https://docs.openshift.com/container-platform/4.13/monitoring/enabling-monitoring-for-user-defined-projects.html) should be enabled
* a [ServiceMonitor](https://docs.openshift.com/container-platform/4.13/monitoring/managing-metrics.html#specifying-how-a-service-is-monitored_managing-metrics) resource should be available

Example of a `ServiceMonitor`:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    k8s-app: prometheus-example-monitor
  name: prometheus-example-monitor
spec:
  endpoints:
  - interval: 30s
    port: otel-8889
    scheme: http
  selector:
    matchLabels:
      otel: ""
```

## Configure Pulp CR to enable telemetry

To enable telemetry, configure Pulp operator CR with the following fields:
```yaml
...
spec:
  telemetry:
    enabled: true
...
```
