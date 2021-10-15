Postgres
========

A role to setup postgres in Pulp 3, yielding the following objects:

* StatefulSet
* Service
* Secret
    * Stores the DB password

Role Variables
--------------

* `database_connection`: A dictionary of database configuration
    * `username`: User that owns and runs Postgres.
    * `password`: Database password.
    * `admin_password`: Initial password for the Pulp admin.
    * `sslmode` is valid for `external` databases only. The allowed values are: `prefer`, `disable`, `allow`, `require`, `verify-ca`, `verify-full`.

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
