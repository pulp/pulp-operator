Pulp Web
========

A role to setup Pulp 3 NGINX web, yielding the following objects:

* Deployment
* Service
* ConfigMap
    *  Stores NGINX configs

Role Variables
--------------

* `web`: A dictionary of pulp-web configuration
    * `replicas`: Number of pod replicas.
* `image_web`: The image name. Default: quay.io/pulp/pulp-web
* `image_web_version`: The image tag. Default: stable
* `nginx_client_max_body_size`: Sets the maximum allowed size of the client request body.

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
