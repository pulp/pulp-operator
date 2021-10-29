Redis
=====

A role to setup Pulp 3 redis, yielding the following objects:

* Deployment
* Service
* PersistentVolumeClaim

Role Variables
--------------

* `redis_image`: The redis image name. Default: redis:latest

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
