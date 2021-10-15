Backup
========

The purpose of this role is to create a backup of your Pulp deployment.  This includes:
  - backup of the PostgreSQL database
  - custom user config file

Role Variables
--------------

* `deployment_name`: The name of the pulp custom resource to backup
* `backup_pvc`: The name of the PVC to uses for backup
* `backup_storage_requirements`: The size of storage for the PVC created by operator if one is not supplied
* `backup_storage_class`: The storage class to be used for the backup PVC
* `postgres_configuration_secret`: The postgres_configuration_secret

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
