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
