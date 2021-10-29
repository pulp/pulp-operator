Restore
========

The purpose of this role is to restore your Pulp deployment from a backup.  This includes:
  - backup of the PostgreSQL database
  - custom user config file

Role Variables
--------------

* `backup_name`: The name of the pulp backup custom resource to restore from
* `postgres_label_selector`: The label selector for an external container based database

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
