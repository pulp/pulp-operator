Pulp Web
========

A role to setup Pulp 3's NGINX web proxy service.

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Role Variables
--------------

* `web`: A dictionary of pulp-web configuration
    * `replicas`: Number of pod replicas.
* `registry`: The container registry.
* `project`: The project name e.g. user or org name at the container registry.
* `image`: The image name.
* `tag`: The tag name.

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
