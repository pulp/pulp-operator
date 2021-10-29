Pulp Worker
===========

A role to setup Pulp 3 worker, yielding the following objects:

* Deployment

Role Variables
--------------

* `worker`: A dictionary of pulp-worker configuration
    * `replicas`: Number of pod replicas.
* `image`: The image name. Default: quay.io/pulp/pulp
* `image_version`: The image tag. Default: stable

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Dependencies
------------

collections:

  - kubernetes.core
  - operator_sdk.util

License
-------

GPLv2+

Author Information
------------------

[Pulp Team](https://pulpproject.org/)
