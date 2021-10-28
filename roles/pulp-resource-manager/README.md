Pulp Resource Manager
=====================

A role to setup Pulp 3 Resource Manager, yielding the following objects:

* Deployment

Role Variables
--------------

* `resource_manager`: A dictionary of pulp-resource-manager configuration
    * `replicas`: Number of pod replicas.
* `image`: The image name. Default: quay.io/pulp/pulp
* `image_version`: The image tag. Default: stable

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Dependencies
------------

collections:

  - community.kubernetes
  - operator_sdk.util

License
-------

GPLv2+

Author Information
------------------

[Pulp Team](https://pulpproject.org/)
