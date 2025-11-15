# Horizontal Pod Autoscaler (HPA) Support

The Pulp Operator now supports Horizontal Pod Autoscaler (HPA) for automatic scaling of Pulp components based on resource utilization.

## Overview

HPA automatically scales the number of pods in a deployment based on observed CPU and/or memory utilization. This feature is available for the following Pulp components:

- **API** (`pulpcore-api`)
- **Content** (`pulpcore-content`)
- **Worker** (`pulpcore-worker`)
- **Web** (`pulpcore-web`)

## Configuration

HPA can be configured independently for each component in the Pulp Custom Resource.

### Basic Example

```yaml
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: example-pulp
spec:
  api:
    replicas: 2  # Ignored when HPA is enabled
    hpa:
      enabled: true
      min_replicas: 2
      max_replicas: 10
      target_cpu_utilization_percentage: 70
```

### Complete Example with All Components

```yaml
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: example-pulp
spec:
  # API component with HPA
  api:
    replicas: 2
    resource_requirements:
      requests:
        cpu: "500m"
        memory: "512Mi"
      limits:
        cpu: "1000m"
        memory: "1Gi"
    hpa:
      enabled: true
      min_replicas: 2
      max_replicas: 10
      target_cpu_utilization_percentage: 70
      target_memory_utilization_percentage: 80

  # Content component with HPA
  content:
    replicas: 2
    resource_requirements:
      requests:
        cpu: "500m"
        memory: "512Mi"
    hpa:
      enabled: true
      min_replicas: 2
      max_replicas: 8
      target_cpu_utilization_percentage: 75

  # Worker component with HPA
  worker:
    replicas: 2
    resource_requirements:
      requests:
        cpu: "500m"
        memory: "512Mi"
    hpa:
      enabled: true
      min_replicas: 1
      max_replicas: 20
      target_cpu_utilization_percentage: 80

  # Web component with HPA (only when not using Route/Ingress)
  web:
    replicas: 2
    resource_requirements:
      requests:
        cpu: "200m"
        memory: "256Mi"
    hpa:
      enabled: true
      min_replicas: 2
      max_replicas: 5
      target_cpu_utilization_percentage: 70
```

## HPA Configuration Fields

### `enabled`
- **Type**: Boolean
- **Default**: `false`
- **Description**: Enables or disables HPA for the component

### `min_replicas`
- **Type**: Integer
- **Default**: `1`
- **Minimum**: `1`
- **Description**: Minimum number of replicas. HPA will not scale below this value.

### `max_replicas`
- **Type**: Integer
- **Required**: Yes (when HPA is enabled)
- **Minimum**: `1`
- **Description**: Maximum number of replicas. HPA will not scale above this value.

### `target_cpu_utilization_percentage`
- **Type**: Integer
- **Optional**: Yes
- **Range**: `1-100`
- **Default**: `50` (if no metrics are specified)
- **Description**: Target average CPU utilization across all pods (as a percentage of requested CPU)

### `target_memory_utilization_percentage`
- **Type**: Integer
- **Optional**: Yes
- **Range**: `1-100`
- **Description**: Target average memory utilization across all pods (as a percentage of requested memory)

## Important Considerations

### 1. Resource Requests are Required

For HPA to work properly, you **must** define resource requests for CPU and/or memory:

```yaml
api:
  resource_requirements:
    requests:
      cpu: "500m"      # Required for CPU-based autoscaling
      memory: "512Mi"  # Required for memory-based autoscaling
  hpa:
    enabled: true
    max_replicas: 10
    target_cpu_utilization_percentage: 70
```

### 2. Replicas Field is Ignored

When HPA is enabled, the `replicas` field is ignored. The HPA controller manages the replica count based on the observed metrics.

### 3. Default Metrics

If neither `target_cpu_utilization_percentage` nor `target_memory_utilization_percentage` is specified, HPA defaults to:
- **CPU**: 50% utilization

### 4. Multiple Metrics

You can specify both CPU and memory targets. HPA will scale based on whichever metric requires more replicas:

```yaml
hpa:
  enabled: true
  max_replicas: 10
  target_cpu_utilization_percentage: 70
  target_memory_utilization_percentage: 80
```

### 5. Metrics Server Required

HPA requires the Kubernetes Metrics Server to be installed in your cluster. Verify it's running:

```bash
kubectl get deployment metrics-server -n kube-system
```

## Monitoring HPA

### Check HPA Status

```bash
# List all HPAs
kubectl get hpa -n <namespace>

# Describe specific HPA
kubectl describe hpa example-pulp-api -n <namespace>
```

### Example HPA Output

```
NAME                REFERENCE                      TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
example-pulp-api    Deployment/example-pulp-api    45%/70%   2         10        3          5m
```

### View HPA Events

```bash
kubectl get events -n <namespace> --field-selector involvedObject.name=example-pulp-api
```

## Disabling HPA

To disable HPA for a component, set `enabled: false` or remove the `hpa` section:

```yaml
api:
  replicas: 3  # This will now be used
  hpa:
    enabled: false
```

The operator will automatically delete the HPA resource and revert to using the static `replicas` value.

## Best Practices

1. **Start Conservative**: Begin with higher target utilization percentages (70-80%) and adjust based on observed behavior

2. **Set Appropriate Min/Max**:
   - `min_replicas`: Should handle baseline load
   - `max_replicas`: Should be based on cluster capacity and cost considerations

3. **Monitor Scaling Behavior**: Watch for:
   - Frequent scaling up/down (thrashing)
   - Hitting max replicas frequently (may need to increase)
   - Staying at min replicas (may be over-provisioned)

4. **Resource Requests**: Set realistic resource requests based on actual usage patterns

5. **Combine with PDB**: Use PodDisruptionBudget to ensure availability during scaling events:

```yaml
api:
  hpa:
    enabled: true
    min_replicas: 3
    max_replicas: 10
  pdb:
    maxUnavailable: 1
```

## Troubleshooting

### HPA Shows "unknown" for Metrics

**Cause**: Metrics Server is not installed or not working

**Solution**:
```bash
# Check Metrics Server
kubectl get apiservice v1beta1.metrics.k8s.io -o yaml

# Install Metrics Server (if needed)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

### HPA Not Scaling

**Possible causes**:
1. Resource requests not defined
2. Metrics Server not running
3. Target utilization already met
4. Insufficient cluster resources

**Debug**:
```bash
# Check HPA status
kubectl describe hpa <hpa-name> -n <namespace>

# Check pod metrics
kubectl top pods -n <namespace>

# Check HPA controller logs
kubectl logs -n kube-system -l k8s-app=kube-controller-manager
```

### Pods Not Scaling Down

HPA has a default cooldown period:
- **Scale up**: 3 minutes
- **Scale down**: 5 minutes

This prevents rapid fluctuations. Wait for the cooldown period before expecting scale-down events.

## Example Scenarios

### Scenario 1: High-Traffic API

```yaml
api:
  resource_requirements:
    requests:
      cpu: "1000m"
      memory: "1Gi"
  hpa:
    enabled: true
    min_replicas: 3
    max_replicas: 20
    target_cpu_utilization_percentage: 70
```

### Scenario 2: Batch Processing Workers

```yaml
worker:
  resource_requirements:
    requests:
      cpu: "500m"
      memory: "512Mi"
  hpa:
    enabled: true
    min_replicas: 1
    max_replicas: 50
    target_cpu_utilization_percentage: 80
```

### Scenario 3: Memory-Intensive Content Serving

```yaml
content:
  resource_requirements:
    requests:
      cpu: "500m"
      memory: "1Gi"
  hpa:
    enabled: true
    min_replicas: 2
    max_replicas: 10
    target_memory_utilization_percentage: 75
```
