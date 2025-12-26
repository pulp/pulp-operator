# Certificate injection in Pulp containers

Pulp operator supports mounting trusted CA certificates into Pulp containers on both OpenShift and vanilla Kubernetes.

## OpenShift

In OpenShift environments, it is possible to [mount additional trust bundles](https://docs.openshift.com/container-platform/4.10/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki) into Pulp containers.

When `mount_trusted_ca: true`, Pulp operator will automatically create and mount a `ConfigMap` with the custom CA into Pulp pods. Before enabling this, users need to follow the steps from [Enabling the cluster-wide proxy](https://docs.openshift.com/container-platform/4.10/networking/configuring-a-custom-pki.html#nw-proxy-configure-object_configuring-a-custom-pki) to register the custom CA certificate into the cluster.

!!! info

    It is recommended to execute the previous steps in a maintenance window because, since this is cluster-wide modification, the cluster can get unavailable if executed wrong (some cluster operators pods will be restarted).

## Vanilla Kubernetes

On vanilla Kubernetes, you can mount CA certificates from a ConfigMap. The ConfigMap can be managed manually or automatically using cert-manager's trust-manager.

See the [CA Certificate Management guide](../../../trust-manager-integration.md) for detailed configuration instructions.
