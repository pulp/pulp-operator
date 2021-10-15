Pulp Content
============

A role to setup content serving in Pulp 3, yielding the following objects:

* Deployment
* Service

Role Variables
--------------

* `content`: A dictionary of pulp-content configuration
    * `replicas`: Number of pod replicas.
    * `log_level`: The desired log level.
* `image`: The image name. Default: quay.io/pulp/pulp:stable

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
