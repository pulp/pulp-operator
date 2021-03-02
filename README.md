![Pulp CI](https://github.com/pulp/pulp-operator/workflows/Pulp%20CI/badge.svg)

# Pulp Operator

A Kubernetes Operator for Pulp 3, under active development (not production ready yet) by the Pulp team. The goal is to provide a scalable and robust cluster for Pulp 3. [Pre-built images are hosted on quay.io](https://quay.io/repository/pulp/pulp-operator).

Note that it utilizes a single container image from the pulpcore repo, to run 4 different types of service containers (like pulpcore-api & pulpcore-content.) currently manually built and [hosted on quay.io](https://quay.io/repository/pulp/pulp).

It is currently working towards [Phase 1 of the Kubernetes Operator Capability Model](https://blog.openshift.com/top-kubernetes-operators-advancing-across-the-operator-capability-model/) before being published on OperatorHub, including compatibility with more clusters.

See [latest slide deck](http://people.redhat.com/mdepaulo/presentations/Introduction%20to%20pulp-operator.pdf) for more info.

## Services

- **pulpcore-api** - serves REST API, Galaxy APIs (v1, v2, v3, UI), and the container registry API. The number of instances of this service should be scaled as demand requires.  Administrators and users of all of the APIs create demand for this service.


- **pulpcore-content** - serves content to clients. pulpcore-api redirects clients here to download content. When content is being mirrored from a remote source this service can download that content and stream it to the client the first time the content is requested. The number of instances of this service should be scaled as demand requires. Content consumers create demand for this service.


- **pulpcore-worker** - performs syncing, importing of content, and other asynchronous operations that required resource locking. The number of instances of this service should be scaled as demand requires. Administrators and content importers create demand for this service.


- **pulpcore-resource-manager** - all asynchronous work flows through this service. Only a single entity does work, but other instances can be run as hot spares that will take over if the active one fails.

## Created with (based on template)
`operator-sdk new pulp-operator --api-version=pulpproject.org/v1beta1 --kind=Pulp --type=ansible --generate-playbook`

## Built/pushed with
`operator-sdk build --image-builder=buildah quay.io/pulp/pulp-operator:latest`

`podman login quay.io`

`podman push quay.io/pulp/pulp-operator:latest`

## Usage

Review `deploy/crds/pulpproject_v1beta1_pulp_cr.default.yaml`. If the variables' default values are not correct for your environment, copy to `deploy/crds/pulpproject_v1beta1_pulp_cr.yaml`, uncomment "spec:", and uncomment and adjust the variables.

`./up.sh`

`minikube service list`

or

Get external ports:

`kubectl get services`

Get external IP addresses:

`kubectl get pods -o wide`


# How to File an Issue

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


# Get Help

Documentation: https://pulp-operator.readthedocs.io/en/latest/

Issue Tracker: https://pulp.plan.io

User mailing list: https://www.redhat.com/mailman/listinfo/pulp-list

Developer mailing list: https://www.redhat.com/mailman/listinfo/pulp-dev

User questions welcome in #pulp on FreeNode IRC server.

Developer discussion in #pulp-dev on FreeNode IRC server
