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

Pulp Operator is under active development and not production ready yet. The goal is to provide a scalable and robust cluster for Pulp 3.

Note that Pulp operator works with three different types of service containers (the operator itself, the main service and the web service):

|           | Operator | Main | Web |
| --------- | -------- | ---- | --- |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[pulp](https://quay.io/repository/pulp/pulp?tab=tags) | [pulp-web](https://quay.io/repository/pulp/pulp-web?tab=tags) |
| **Image** | [pulp-operator](https://quay.io/repository/pulp/pulp-operator?tab=tags) |[galaxy](https://quay.io/repository/pulp/galaxy?tab=tags) | [galaxy-web](https://quay.io/repository/pulp/galaxy-web?tab=tags) |

<br>Pulp operator is manually built and [hosted on quay.io](https://quay.io/repository/pulp/pulp-operator). Read more about the container images [here](https://docs.pulpproject.org/pulp_operator/container/).

## Get Help

Documentation: [https://docs.pulpproject.org/pulp_operator/](https://docs.pulpproject.org/pulp_operator/)

Issue Tracker: [https://pulp.plan.io](https://pulp.plan.io)

Forum: [https://discourse.pulpproject.org/](https://discourse.pulpproject.org/)

Join [**#pulp** on Matrix](https://matrix.to/#/#pulp:matrix.org)

Join [**#pulp-dev** on Matrix](https://matrix.to/#/#pulp-dev:matrix.org) for Developer discussion.

## How to File an Issue

To file a new issue set the Category to `Operator` when filing [here](https://pulp.plan.io/projects/pulp/issues/new).

See [redmine fields](https://docs.pulpproject.org/bugs-features.html#redmine-fields) for more detailed
descriptions of all the fields and how they are used.

| Field | Instructions |
| ----- | ----------- |
| Tracker | For a bug, select `Issue`, for a feature-request, choose `Story` |
| Subject | Strive to be specific and concise. |
| Description | This is the most important part! Please see [issue description](https://docs.pulpproject.org/bugs-features.html#issue-description). |
| Category | Operator |
| Version | The version of operator that you discovered the issue. |
| OS | The Ansible managed OS. |
