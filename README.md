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
* Documentation: [https://docs.pulpproject.org/](https://docs.pulpproject.org/)
* Source: [https://github.com/pulp/pulpcore/](https://github.com/pulp/pulpcore/)
* Bugs: [https://pulp.plan.io/projects/pulp](https://pulp.plan.io/projects/pulp)

For more information, check out the project website: [https://pulpproject.org](https://pulpproject.org)

If you want to evaluate Pulp quickly, try [Pulp in One Container](https://pulpproject.org/pulp-in-one-container/)

## Pulp Operator

An [Ansible Operator](https://www.ansible.com/blog/ansible-operator) for Pulp 3.

Pulp Operator is under active development, with the goal to provide a scalable and robust cluster for Pulp 3.

Note that Pulp operator works with three different types of service containers (the operator itself, the main service and the web service):

|           | Operator | Main | Web |
| --------- | -------- | ---- | --- |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[pulp](https://quay.io/repository/pulp/pulp?tab=tags) | [pulp-web](https://quay.io/repository/pulp/pulp-web?tab=tags) |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[galaxy](https://quay.io/repository/pulp/galaxy?tab=tags) | [galaxy-web](https://quay.io/repository/pulp/galaxy-web?tab=tags) |

<br>Pulp operator is manually built and [hosted on quay.io](https://quay.io/repository/pulp/pulp-operator). Read more about the container images [here](https://docs.pulpproject.org/pulp_operator/container/).

## Custom Resource Definitions
Pulp Operator currently provides three different kinds of [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-resources): Pulp, Pulp Backup and Pulp Restore.
### Pulp
Manages the Pulp application and its deployments, services, etc. Through the following ansible roles:

* [API](https://docs.pulpproject.org/pulp_operator/roles/pulp-api/)
* [Content](https://docs.pulpproject.org/pulp_operator/roles/pulp-content/)
* [Routes](https://docs.pulpproject.org/pulp_operator/roles/pulp-routes/)
* [Worker](https://docs.pulpproject.org/pulp_operator/roles/pulp-worker/)
* [Web](https://docs.pulpproject.org/pulp_operator/roles/pulp-web/)
* [Status](https://docs.pulpproject.org/pulp_operator/roles/pulp-status/)
* [Postgres](https://docs.pulpproject.org/pulp_operator/roles/postgres/)
* [Redis](https://docs.pulpproject.org/pulp_operator/roles/redis/)

### Pulp Backup
Manages pulp backup through the following ansible role:

* [Backup](https://docs.pulpproject.org/pulp_operator/roles/backup/)

### Pulp Restore
Manages the restoration of a pulp backup through the following ansible role:

* [Restore](https://docs.pulpproject.org/pulp_operator/roles/restore/)
## Get Help

Documentation: [https://docs.pulpproject.org/pulp_operator/](https://docs.pulpproject.org/pulp_operator/)

Issue Tracker: [https://github.com/pulp/pulp-operator/issues](https://github.com/pulp/pulp-operator/issues)

Forum: [https://discourse.pulpproject.org/](https://discourse.pulpproject.org/)

Join [**#pulp** on Matrix](https://matrix.to/#/#pulp:matrix.org)

Join [**#pulp-dev** on Matrix](https://matrix.to/#/#pulp-dev:matrix.org) for Developer discussion.
