Pulp API
========

A role to setup Pulp 3's API service.

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Role Variables
--------------

* `api`: A dictionary of pulp-api configuration
    * `replicas`: Number of pod replicas.
    * `log_level`: The desired log level.
* `default_settings`: A nested dictionary that will be combined with custom values from the user's
    `setting.py`. The keys of this dictionary are variable names, and the values should be expressed using the
    [Dynaconf syntax](https://dynaconf.readthedocs.io/en/latest/guides/environment_variables.html#precedence-and-type-casting)
    Please see [pulpcore configuration docs](https://docs.pulpproject.org/en/master/nightly/installation/configuration.html#id2)
    for documentation on the possible variable names and their values.
    * `debug`: Wether to run pulp in debug mode.
* `registry`: The container registry.
* `project`: The project name e.g. user or org name at the container registry.
* `image`: The image name.
* `tag`: The tag name.
* `storage_type`: A string for specifying storage configuration type.
* `file_storage`: A dict for specifying a persistent volume claim for pulp-file.
    * `access_mode`: The access mode for the volume.
    * `size`: The storage size.
* `object_storage_s3_secret`:The kubernetes secret with s3 storage configuration information.
* `object_storage_azure_secret`:The kubernetes secret with Azure blob storage configuration information.

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
