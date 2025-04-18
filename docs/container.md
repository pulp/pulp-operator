# Containers

## Pulp

An all-in-one [pulp](https://github.com/pulp/pulpcore) image that can assume each of the following types of service:

- **pulpcore-api** - serves the Pulp(v3) API. The number of instances of this service should be scaled as demand requires.  _Administrators and users of all of the APIs put demand on this service_.


- **pulpcore-content** - serves content to clients. pulpcore-api redirects clients to pulpcore-content to download content. When content is being mirrored from a remote source, this service can download that content and stream it to the client the first time the content is requested. The number of instances of this service should be scaled as demand requires. _Content consumers put demands on this service_.


- **pulpcore-worker** - performs syncing, importing of content, and other asynchronous operations that require resource locking. The number of instances of this service should be scaled as demand requires. _Administrators and content importers put demands on this service_.


Currently built with the following plugins:

* [pulp_ansible](https://docs.pulpproject.org/pulp_ansible/)
* [pulp-certguard](https://docs.pulpproject.org/pulp_certguard/)
* [pulp_container](https://docs.pulpproject.org/pulp_container/)
* [pulp_deb](https://docs.pulpproject.org/pulp_deb/)
* [pulp_file](https://docs.pulpproject.org/pulp_file/)
* [pulp_python](https://docs.pulpproject.org/pulp_python/)
* [pulp_rpm](https://docs.pulpproject.org/pulp_rpm/)

### Tags

* `latest`: Built nightly, with master/main branches of each plugin.
* `stable`: Built on push, with latest released version of each plugin.
* `3.y.z`:  Pulpcore 3.y.z version and its compatible plugins.

[https://quay.io/repository/pulp/pulp?tab=tags](https://quay.io/repository/pulp/pulp?tab=tags)


## Pulp Web

An Nginx image based on [centos/nginx-116-centos7](https://hub.docker.com/r/centos/nginx-116-centos7),
with pulpcore and plugins specific configuration.

### Tags

* `latest`: Built nightly, with master/main branches of [pulpcore](https://github.com/pulp/pulpcore) and its plugins.
* `stable`: Built on push, with latest released version of each plugin.
* `3.y.z`:  Pulpcore 3.y.z version and its compatible plugins.

[https://quay.io/repository/pulp/pulp-web?tab=tags](https://quay.io/repository/pulp/pulp-web?tab=tags)



## Pulp Operator

An image with the pulp operator binary.

### Tags

* `latest`: Built nightly, with main branch of [pulp-operator](https://github.com/pulp/pulp-operator).
* `0.y.z`:  Pulp Operator 0.y.z version.

[https://quay.io/repository/pulp/pulp-operator?tab=tags](https://quay.io/repository/pulp/pulp-operator?tab=tags)


## Build

The images can be found [here](https://github.com/pulp/pulp-oci-images/tree/latest/images): [https://docs.pulpproject.org/pulp_oci_images/](https://docs.pulpproject.org/pulp_oci_images/)
