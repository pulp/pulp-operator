# Sample Configs

To help with the first steps of Pulp installation and [CR definition](https://pulpproject.org/pulp-operator/docs/admin/reference/custom_resources/repo_manager/#pulp), there are a set of [samples in pulp-operator repository](https://github.com/pulp/pulp-operator/tree/main/config/samples).



### MINIMAL CONFIGURATION

The [`minimal.yaml`](https://github.com/pulp/pulp-operator/blob/main/config/samples/minimal.yaml) file, provides a sample CR with
the minimal configurations needed (the [database](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/database/) and [pulpcore storage](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/) definitions) for the operator to run. It will deploy Pulp with a single [replica of pulpcore](https://pulpproject.org/pulp-operator/docs/admin/guides/install/ha/#scaling-pulpcore-pods) pods and a [random password for the admin user](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/reset_admin_pwd/#reset-pulp-admin-password).

!!! note

    The `minimal.yaml` sample will not deploy a reverse proxy or expose pulpcore pods.


### SIMPLE

The [`simple.yaml`](https://github.com/pulp/pulp-operator/blob/main/config/samples/simple.yaml) file, provides a sample CR
that will deploy Pulp with:

- a single [replica of pulpcore](https://pulpproject.org/pulp-operator/docs/admin/guides/install/ha/#scaling-pulpcore-pods) pods
- a definition of the [StorageClass to store Pulp artifacts](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-pulp-operator-storage-to-use-a-storage-class)
- a [database](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/database/) pod
- a [cache](http://127.0.0.1:8000/pulp-operator/docs/admin/guides/configurations/cache/#configure-pulp-operator-to-deploy-a-redis-instance) pod
- a [k8s Service type nodeport](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/exposing/#nodeport)
- a [pre-defined password for the admin user](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/secrets/#pulp-admin-password)
- some [Pulp app configs](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/pulp_settings/#custom-settings) (settings.py definitions)


### S3

The [`simple.s3.ci.yaml`](https://github.com/pulp/pulp-operator/blob/main/config/samples/simple.s3.ci.yaml) file, provides a sample CR
that will deploy Pulp with:

- a single [replica of pulpcore](https://pulpproject.org/pulp-operator/docs/admin/guides/install/ha/#scaling-pulpcore-pods) pods
- an [object storage (S3) to store Pulp artifacts](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-aws-s3-storage)
- a [database](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/database/) pod
- a [k8s Service type nodeport](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/exposing/#nodeport)
- a [pre-defined password for the admin user](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/secrets/#pulp-admin-password)
- some [Pulp app configs](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/pulp_settings/#custom-settings) (settings.py definitions)


### EXTERNAL DATABASE

The [`simple-external-db.yaml`](https://github.com/pulp/pulp-operator/blob/main/config/samples/simple-external-db.yaml) file, provides a sample CR
that will deploy Pulp with:

- a single [replica of pulpcore](https://pulpproject.org/pulp-operator/docs/admin/guides/install/ha/#scaling-pulpcore-pods) pods
- a definition of the [StorageClass to store Pulp artifacts](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-pulp-operator-storage-to-use-a-storage-class)
- a definition of a [k8s Secret with the credentials to access a postgres database](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/database/#configure-pulp-operator-to-use-an-external-postgresql-installation) (not deployed/managed by pulp-operator)
- a [k8s Service type nodeport](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/exposing/#nodeport)
- a [pre-defined password for the admin user](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/secrets/#pulp-admin-password)
- some [Pulp app configs](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/pulp_settings/#custom-settings) (settings.py definitions)


### INGRESS


The [`simple.ingress.yaml`](https://github.com/pulp/pulp-operator/blob/main/config/samples/simple.ingress.yaml) file, provides a sample CR
that will deploy Pulp with:

- a single [replica of pulpcore](https://pulpproject.org/pulp-operator/docs/admin/guides/install/ha/#scaling-pulpcore-pods) pods
- a definition of the [StorageClass to store Pulp artifacts](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-pulp-operator-storage-to-use-a-storage-class)
- a [database](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/database/) pod
- some [Pulp app configs](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/pulp_settings/#custom-settings) (settings.py definitions)
- a [k8s ingress](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/exposing/#ingress)
