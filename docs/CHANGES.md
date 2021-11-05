Changelog
=========

<!---
    You should *NOT* be adding new change log entries to this file, this
    file is managed by towncrier. You *may* edit previous change logs to
    fix problems like typo corrections or such.
    To add a new change log entry, please see
    https://docs.pulpproject.org/contributing/git.html#changelog-update

    WARNING: Don't drop the next directive!
--->

<!-- TOWNCRIER -->

0.5.0 (2021-11-05)
==================


Features
--------

- Made request size limit configurable
  [#227](https://github.com/pulp/pulp-operator/issues/227)
- Ensure resource manager is not started for pulpcore >= 3.16
  [#231](https://github.com/pulp/pulp-operator/issues/231)
- Set RELATED_IMAGE_ vars to enable disconnected deployments
  [#232](https://github.com/pulp/pulp-operator/issues/232)


Bugfixes
--------

- Image pull policy defaults to IfNotPresent
  [#229](https://github.com/pulp/pulp-operator/issues/229)


----


0.4.0 (2021-10-15)
==================


Features
--------

- Removed tags, registry, and projects so users can add images with custom registries and tags in image override
  [#218](https://github.com/pulp/pulp-operator/issues/218)
- Create or import a key for pulp-api to use when encrypting sensitive db fields
  [#8730](https://pulp.plan.io/issues/8730)
- Enable new tasking system
  [#9020](https://pulp.plan.io/issues/9020)
- Added support to override PosgreSQL sslmode
  [#9421](https://pulp.plan.io/issues/9421)


Bugfixes
--------

- Ensure default storage for Postgres
  [#221](https://github.com/pulp/pulp-operator/issues/221)


Deprecations and Removals
-------------------------

- Move from cluster-scoped operator model to namespace-scoped model
  [#208](https://github.com/pulp/pulp-operator/issues/208)
- Dropping OCP 4.6 support
  [#9330](https://pulp.plan.io/issues/9330)


Misc
----

- [#206](https://github.com/pulp/pulp-operator/issues/206), [#209](https://github.com/pulp/pulp-operator/issues/209), [#215](https://github.com/pulp/pulp-operator/issues/215), [#9217](https://pulp.plan.io/issues/9217)


----


0.3.0 (2021-07-14)
==================


Features
--------

- Enable container based database migration support
  [#8472](https://pulp.plan.io/issues/8472)
- Enable backup of database and secrets associated with Pulp custom resource
  [#8473](https://pulp.plan.io/issues/8473)
- Enable backup of storage associated with Pulp custom resource
  [#8474](https://pulp.plan.io/issues/8474)
- Enable restore of deployment associated with Pulp custom resource backup
  [#8513](https://pulp.plan.io/issues/8513)
- Add additional backup and restore flexibility to allow for restore from only a PVC
  [#8630](https://pulp.plan.io/issues/8630)
- Allow user to specify the storage class for the Redis PVC
  [#8877](https://pulp.plan.io/issues/8877)


Bugfixes
--------

- Allow user to specify empty string for PostgreSQL PVC storage class
  [#8733](https://pulp.plan.io/issues/8733)
- Update nodeport templating in API and Content services
  [#8810](https://pulp.plan.io/issues/8810)
- Fix collision on file_storage fact usage after pulp prefix cleanup
  [#8832](https://pulp.plan.io/issues/8832)
- Fix Nodeport flow to create ports in standard range and only on the web service. Also allows node_ip discover based on where the pod is running.
  [#8833](https://pulp.plan.io/issues/8833)
- Resolve Pulp status correctly when deployed in a separate namespace
  [#8880](https://pulp.plan.io/issues/8880)


Improved Documentation
----------------------

- Document how to deploy Pulp on OpenShift
  [#8836](https://pulp.plan.io/issues/8836)


Misc
----

- [#8530](https://pulp.plan.io/issues/8530), [#8563](https://pulp.plan.io/issues/8563), [#8598](https://pulp.plan.io/issues/8598)


----


0.2.0 (2021-03-26)
==================


Features
--------

- Add deployment of nginx webserver with pulp snippets
  [#5657](https://pulp.plan.io/issues/5657)
- Container building machinery for the operator
  [#7171](https://pulp.plan.io/issues/7171)
- Enable the creation of Ingress or Route objects based on the specifications within the custom resource
  [#8272](https://pulp.plan.io/issues/8272)
- Deploy postgres database using a secret to store configuration instead of it existing in the custom resource; allows credentials to be kept secret.
  [#8289](https://pulp.plan.io/issues/8289)
- Enable the use of S3 compliant or Azure object storage as storage backend
  [#8361](https://pulp.plan.io/issues/8361)
- Operator will provide information data via custom resource status object
  [#8402](https://pulp.plan.io/issues/8402)
- Enable installation of operator using OLM catalog
  [#8409](https://pulp.plan.io/issues/8409)
- Enable resource requirement specification for deployments and have operator check for running nodes and healthy status
  [#8456](https://pulp.plan.io/issues/8456)


Bugfixes
--------

- Only build plugins from pulp org
  [#7234](https://pulp.plan.io/issues/7234)
- Fix storage option check so that Azure Blob Storage can be used as a backend
  [#8424](https://pulp.plan.io/issues/8424)


Misc
----

- [#5134](https://pulp.plan.io/issues/5134), [#5141](https://pulp.plan.io/issues/5141), [#5142](https://pulp.plan.io/issues/5142), [#7107](https://pulp.plan.io/issues/7107), [#8273](https://pulp.plan.io/issues/8273), [#8293](https://pulp.plan.io/issues/8293), [#8294](https://pulp.plan.io/issues/8294), [#8345](https://pulp.plan.io/issues/8345), [#8353](https://pulp.plan.io/issues/8353), [#8370](https://pulp.plan.io/issues/8370), [#8378](https://pulp.plan.io/issues/8378), [#8399](https://pulp.plan.io/issues/8399), [#8418](https://pulp.plan.io/issues/8418)


----
