![Pulp CI](https://github.com/pulp/pulp-operator/workflows/Pulp%20CI/badge.svg)

# Pulp

[Pulp](https://pulpproject.org/) is a platform for managing repositories of content, such as software packages, and making them available to a large number of consumers.

With Pulp you can:

* Locally mirror all or part of a repository
* Host your own content in a new repository
* Manage content from multiple sources in one place
* Promote content through different repos in an organized way

If you have dozens, hundreds, or thousands of software packages and need a better way to manage them, Pulp can help.

Pulp is completely free and open-source!

* License: GPLv2+
* Source: [https://github.com/pulp/pulpcore/](https://github.com/pulp/pulpcore/)

For more information, check out the project website: [https://pulpproject.org](https://pulpproject.org)

If you want to evaluate Pulp quickly, try [Pulp in One Container](https://pulpproject.org/pulp-in-one-container/)


## Kubernetes Operators

Kubernetes operators allows to manage services in an automated way by implementing the human operator knowledge
into code. More information: [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

## Pulp Operator

Pulp Operator is in beta stage and under active development, with the goal to provide a scalable and robust cluster for Pulp 3.
If you find any problem, please check our [issue page](https://github.com/pulp/pulp-operator/issues?q=is%3Aissue+is%3Aopen+label%3Ago-alpha) with the current known issues or feel free to fill a new bug or rfe in case it is not addressed yet.

Note that Pulp operator works with three different types of service containers (the operator itself, the main service and the web service):

|           | Operator | Main | Web |
| --------- | -------- | ---- | --- |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[pulp-minimal](https://quay.io/repository/pulp/pulp-minimal?tab=tags) | [pulp-web](https://quay.io/repository/pulp/pulp-web?tab=tags) |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[galaxy-minimal](https://quay.io/repository/pulp/galaxy-minimal?tab=tags) | [galaxy-web](https://quay.io/repository/pulp/galaxy-web?tab=tags) |

<br>Pulp operator is manually built and [hosted on quay.io](https://quay.io/repository/pulp/pulp-operator). Read more about the container images [here](https://docs.pulpproject.org/pulp_operator/container/).


## Getting Started

For a quickstart guide to install and run Pulp Operator check the [Getting Started doc](/pulp_operator/quickstart/#getting-started).

## Custom Resource Definitions
Pulp Operator currently provides three different kinds of [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-resources): Pulp, Pulp Backup and Pulp Restore.
### Pulp
Manages the Pulp application and its deployments, services, etc.

### Pulp Backup
Manages pulp backup
### Pulp Restore
Manages the restoration of a pulp backup
## Get Help

Issue Tracker: [https://github.com/pulp/pulp-operator/issues](https://github.com/pulp/pulp-operator/issues)

Forum: [https://discourse.pulpproject.org/](https://discourse.pulpproject.org/)

Join [**#pulp** on Matrix](https://matrix.to/#/#pulp:matrix.org)

Join [**#pulp-dev** on Matrix](https://matrix.to/#/#pulp-dev:matrix.org) for Developer discussion.

