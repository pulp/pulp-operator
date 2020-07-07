Redis
=====

A role to setup Pulp 3's redis service.

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Role Variables
--------------

* `registry`: The container registry.
* `project`: The project name e.g. user or org name at the container registry.

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
