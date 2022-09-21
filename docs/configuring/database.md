# Configuring Pulp Database

Pulp operator provides a PostgreSQL database for Pulp to use, but it is also possible to configure the operator to use an external PostgreSQL installation. At this time, [Pulp 3.0 will only work with PostgreSQL](https://docs.pulpproject.org/pulpcore/installation/instructions.html?highlight=database#database-setup).


## Configuring Pulp operator to deploy a PostgreSQL instance

[Pulp CR page](/pulp_operator/pulp/#database) has all the parameters that can be set to inform Pulp operator how it should deploy the PostgreSQL container.

If no `database` parameter is defined, Pulp operator will deploy PostgreSQL with the following configuration:

* a `StatefulSet` will be provisioned to handle PostgreSQL pod
* a single PostgreSQL replica will be available (it is **not** possible to form a cluster with this container)
* it will deploy a `docker.io/library/postgres:13` image
* **no data will be persisted**, the container will mount an emptyDir (all data will be lost in case of pod restart)


A new `Secret` (&lt;deployment-name>-postgres-configuration) will also be created with some information like:

  * the database name
  * the admin user
  * the admin password
  * the address to communicate with the database (this is a `k8s svc` address)
  * the service port

A `Service` will be created with the PostgreSQL pod as endpoint.

Here is an example of how to configure Pulp operator to deploy the database using a `Storage Class` called `standard`:
```
...
spec:
  database:
    postgres_storage_class: standard
...
```


## Configuring Pulp operator to use an external PostgreSQL installation

It is also possible to configure Pulp operator to point to a running PostgreSQL cluster.
To do so, create a new `Secret` with the parameters to connect to the running PostgreSQL cluster:
```
$ kubectl -npulp create secret generic external-database \
        --from-literal=POSTGRES_HOST=my-postgres-host.example.com  \
        --from-literal=POSTGRES_PORT=5432  \
        --from-literal=POSTGRES_USERNAME=pulp-admin  \
        --from-literal=POSTGRES_PASSWORD=password  \
        --from-literal=POSTGRES_DB_NAME=pulp \
        --from-literal=POSTGRES_SSLMODE=prefer
```

Make sure to define **all** of the above keys with your cluster configuration.

Now, configure Pulp operator CR to use the Secret:
```
...
spec:
  database:
    external_db_secret: external-database
...
```


!!! warning
    The current version of Pulp backup operator does not support the backup of external databases.
    Only the backup of databases deployed by the operator was tested.