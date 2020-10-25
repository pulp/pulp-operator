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
```
cd pulp-operator
operator-sdk init --domain=pulpproject.org --plugins=ansible
operator-sdk create api --group pulpproject.org --version v1alpha1 --kind Pulp --generate-playbook
```

## Built/pushed with
`make docker-build docker-push IMG=quay.io/pulp/pulp-operator:latest`

## Usage

Review `config/samples/pulp_v1alpha1_pulp.default.yaml`. If the variables' default values are not correct for your environment, copy to `config/samples/pulp_v1alpha1_pulp.yaml`, uncomment "spec:", and uncomment and adjust the variables.

`./up.sh`

`minikube service list`

or

Get external ports:

`kubectl get services`

Get external IP addresses:

`kubectl get pods -o wide`
