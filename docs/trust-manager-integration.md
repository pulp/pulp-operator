# CA Certificate Management

This guide explains how to configure Pulp Operator to mount trusted CA certificates into Pulp pods.

## Overview

Pulp Operator supports two modes for mounting trusted CA certificates into Pulp pods:

1. **OpenShift Mode**: Uses OpenShift's Cluster Network Operator (CNO) to inject CA bundles
2. **ConfigMap Mode**: Mounts CA bundles from a user-specified ConfigMap on vanilla Kubernetes

On vanilla Kubernetes, the ConfigMap can be managed manually or kept up to date automatically using cert-manager's trust-manager.

## Prerequisites

For ConfigMap mode on vanilla Kubernetes, you need:

1. A Kubernetes cluster (non-OpenShift)
2. A ConfigMap containing CA certificates

### Option A: Manual ConfigMap Management

Create a ConfigMap with your CA certificates in the same namespace as your Pulp installation:

```bash
kubectl create configmap my-ca-bundle \
  --from-file=ca.crt=my-ca-bundle.pem \
  --namespace <pulp-namespace>
```

### Option B: Automated Management with trust-manager

For automatic CA bundle updates, install cert-manager and trust-manager:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install trust-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install trust-manager jetstack/trust-manager --namespace cert-manager
```

## Configuration

### Option A: Manual ConfigMap

If managing the ConfigMap manually, configure your Pulp CR in the same namespace as the ConfigMap:

```yaml
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: example-pulp
  namespace: <pulp-namespace>
spec:
  # Enable CA bundle mounting
  mount_trusted_ca: true

  # Specify the ConfigMap and key containing CA certificates
  mount_trusted_ca_configmap_key: "my-ca-bundle:ca.crt"

  # ... other Pulp configuration
```

### Option B: Using trust-manager

#### Step 1: Create a Bundle Resource

Create a `Bundle` resource in the same namespace as your Pulp installation. The Bundle will create a ConfigMap in that namespace:

```yaml
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: example-pulp-trusted-ca-bundle
  namespace: <pulp-namespace>
spec:
  sources:
  # Include default system CAs
  - useDefaultCAs: true

  # Optional: Include custom CAs from ConfigMaps
  # - configMap:
  #     name: my-custom-ca
  #     key: ca.crt

  # Optional: Include CAs from cert-manager Certificates
  # - secret:
  #     name: "my-cert"
  #     key: "ca.crt"

  target:
    configMap:
      key: "ca-bundle.crt"
```

This will create a ConfigMap named `example-pulp-trusted-ca-bundle` containing the aggregated CA bundle.

#### Step 2: Configure Pulp to Use the Bundle

In your Pulp CR, reference the trust-manager ConfigMap:

```yaml
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: example-pulp
  namespace: <pulp-namespace>
spec:
  # Enable CA bundle mounting
  mount_trusted_ca: true

  # Specify the ConfigMap and key (format: "configmap-name:key")
  # The ConfigMap must be in the same namespace as the Pulp CR
  mount_trusted_ca_configmap_key: "example-pulp-trusted-ca-bundle:ca-bundle.crt"

  # ... other Pulp configuration
```

## How It Works

### OpenShift Mode (Automatic)

On OpenShift clusters, when you set `mount_trusted_ca: true`:

1. Operator creates an empty ConfigMap in the Pulp namespace with the label `config.openshift.io/inject-trusted-cabundle: true`
2. OpenShift's CNO automatically injects the cluster's CA bundle into this ConfigMap
3. Operator mounts the ConfigMap at `/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem`

**Note:** You should NOT set `mount_trusted_ca_configmap_key` on OpenShift.

### ConfigMap Mode (Explicit Configuration)

On vanilla Kubernetes clusters:

1. A ConfigMap containing CA certificates exists in the Pulp namespace (created manually or by trust-manager)
2. When both `mount_trusted_ca: true` and `mount_trusted_ca_configmap_key` are set, operator mounts the ConfigMap at `/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem`

**Important:** The ConfigMap must be in the same namespace as the Pulp CR. Cross-namespace ConfigMap references are not supported.

### Using trust-manager (Optional)

If using trust-manager for automated CA bundle management:

1. Trust-manager watches `Bundle` resources
2. Trust-manager aggregates CAs from the specified sources
3. Trust-manager creates/updates the target ConfigMap with the CA bundle
4. Pulp operator mounts the ConfigMap into pods

## ConfigMap Key Format

The `mount_trusted_ca_configmap_key` field uses the format:

```
configmap-name:key
```

For example: `my-ca-bundle:ca.crt` refers to the `ca.crt` key in the `my-ca-bundle` ConfigMap.

## Mount Path

Both modes mount the CA bundle at the same location to ensure compatibility with Red Hat/Fedora-based container images:

```
/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
```

This is the standard system-wide CA bundle location on RHEL/Fedora systems.

## Complete Example

See [config/samples/simple-trust-manager.yaml](../config/samples/simple-trust-manager.yaml) for a complete working example.

## Affected Components

The CA bundle is mounted in all three Pulp core components:

- `pulpcore-api` pods
- `pulpcore-content` pods
- `pulpcore-worker` pods

## Troubleshooting

### ConfigMap Not Found

If you see errors about the ConfigMap not being found:

1. Verify the Bundle resource was created: `kubectl get bundle -n <pulp-namespace>`
2. Check trust-manager logs: `kubectl logs -n cert-manager -l app.kubernetes.io/name=trust-manager`
3. Verify the ConfigMap exists in the Pulp namespace: `kubectl get configmap <configmap-name> -n <pulp-namespace>`
4. Ensure the ConfigMap is in the same namespace as the Pulp CR

### CA Bundle Not Being Used

If applications in Pulp pods are not trusting your CAs:

1. Verify the mount is present in the pods:
   ```bash
   kubectl exec -it -n <pulp-namespace> <pulp-pod> -- ls -la /etc/pki/ca-trust/extracted/pem/
   ```

2. Check the CA bundle content:
   ```bash
   kubectl exec -it -n <pulp-namespace> <pulp-pod> -- cat /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem
   ```

3. Ensure your application is configured to use the system CA bundle (most applications do this by default)

### Specifying Different Keys

The key name in the ConfigMap can be anything:

1. Update the Bundle's `target.configMap.key` field
2. Update the Pulp CR's `mount_trusted_ca_configmap_key` field to match

Example:
```yaml
# Bundle (in pulp namespace)
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: example-pulp-trusted-ca-bundle
  namespace: <pulp-namespace>
spec:
  target:
    configMap:
      key: "custom-bundle.pem"

# Pulp CR (in same namespace)
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: example-pulp
  namespace: <pulp-namespace>
spec:
  mount_trusted_ca_configmap_key: "example-pulp-trusted-ca-bundle:custom-bundle.pem"
```

## Migration from OpenShift to Vanilla Kubernetes

If you're migrating a Pulp instance from OpenShift to vanilla Kubernetes:

1. Install trust-manager on the vanilla Kubernetes cluster
2. Create a Bundle resource with the desired CAs
3. Update the Pulp CR to add `mount_trusted_ca_configmap_key`
4. The existing `mount_trusted_ca: true` field can remain

The operator will automatically detect the presence of `mount_trusted_ca_configmap_key` and switch to trust-manager mode.

## References

- [cert-manager Documentation](https://cert-manager.io/docs/)
- [trust-manager Documentation](https://cert-manager.io/docs/trust/trust-manager/)
- [OpenShift Cluster-wide Proxy Configuration](https://docs.openshift.com/container-platform/latest/networking/enable-cluster-wide-proxy.html)
