Common
========

A role to setup shared tasks in Pulp 3.

In order to pull the images from a private registry (in a disconnected installation, for example) it is
possible to configure the `image_pull_secrets` with the names of the secrets that have the credentials to
pull the images from the private registry.
The secrets that will be used by `image_pull_secrets` need to be created manually:
~~
kubectl create secret docker-registry <name-of-the-secret> --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>
~~

If you have a `~/.docker/config.json` already, you can create the secret through:
~~
kubectl create secret docker-registry <name-of-the-secret> --from-file=.dockerconfigjson=path/to/.docker/config.json
~~

Role Variables
--------------

* `image_pull_secrets`: An array of secrets that will be used to pull image from private registries

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
