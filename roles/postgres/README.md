Postgres
========

A role to setup Pulp 3's postgres service.

Requirements
------------

Requires the `openshift` Python library to interact with Kubernetes: `pip install openshift`.

Role Variables
--------------

* `database_connection`: A dictionary of database configuration
    * `username`: User that owns and runs Postgres.
    * `password`: Database password.
    * `admin_password`: Initial password for the Pulp admin.

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
