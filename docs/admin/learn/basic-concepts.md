# Basic Concepts

## Service Types

Pulp operator works with three different types of service containers: the *Operator* itself, the *Main* service and the *Web* service.

It is manually built and [hosted on quay.io](https://quay.io/repository/pulp/pulp-operator).
If you want to know more about the underlying container images, visit the [Pulp Container Images](site:pulp-operator/docs/admin/reference/container/) section.

|           | Operator | Main | Web |
| --------- | -------- | ---- | --- |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[pulp-minimal](https://quay.io/repository/pulp/pulp-minimal?tab=tags) | [pulp-web](https://quay.io/repository/pulp/pulp-web?tab=tags) |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[galaxy-minimal](https://quay.io/repository/pulp/galaxy-minimal?tab=tags) | [galaxy-web](https://quay.io/repository/pulp/galaxy-web?tab=tags) |

## Custom Resource Definitions

Pulp Operator currently provides three different kinds of [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-resources): Pulp, Pulp Backup and Pulp Restore.

- **Pulp**: Manages the Pulp application and its deployments, services, etc.
- **Pulp Backup**: Manages pulp backup
- **Pulp Restore**: Manages the restoration of a pulp backup

## Architecture

Some components, like `pulp-web`, are not mandatory and depending on how `Pulp CR` is configured
the operator will take care of configuring the other resources that depend on them.

<figure markdown="span">
  ![Pulp Architecture](site:pulp-operator/docs/assets/pulp_architecture.png)
  <figcaption>Overview of a common Pulp Operator installation.</figcaption>
</figure>

## Further Reading

- [Learn more about the Pulp Architecture](site:pulpcore/docs/admin/learn/architecture/).
