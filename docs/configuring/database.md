# Configuring Pulp Database

Pulp operator provides a PostgreSQL database for Pulp to use, but it is also possible to configure the operator to use an external PostgreSQL installation. At this time, [Pulp 3.0 will only work with PostgreSQL](https://docs.pulpproject.org/pulpcore/installation/instructions.html?highlight=database#database-setup).


## Configuring Pulp operator to deploy a PostgreSQL instance

[Pulp CR page](https://docs.pulpproject.org/pulp_operator/pulp/#database) has all the parameters that can be set to inform Pulp operator how it should deploy the PostgreSQL container.

To configure Pulp operator to deploy PostgreSQL, **it is required to define the [storage configurations for the database pod](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#pulp-operator-storage-configuration)**.
Pulp operator will deploy PostgreSQL with the following configuration:

* a `StatefulSet` will be provisioned to handle PostgreSQL pod
* a single PostgreSQL replica will be available (it is **not** possible to form a cluster with this container)
* it will deploy a `docker.io/library/postgres:13` image


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

## Encrypting sensitive fields

Pulp uses a url-safe base64-encoded string of 32 random bytes to encrypt sensitive fields in the database. It is stored as a `Secret` defined in `.spec.db_fields_encryption_secret`. If the `db_fields_encryption_secret` field is not defined during installation, Pulp Operator will create a default one:
```yaml
$ kubectl get pulp -oyaml
...
spec:
  db_fields_encryption_secret: pulp-db-fields-encryption
...


$ kubectl get secret/pulp-db-fields-encryption -oyaml
apiVersion: v1
data:
  database_fields.symmetric.key: RmRWL3...
kind: Secret
metadata:
  name: pulp-db-fields-encryption
...
type: Opaque
```

The key can be generated independently but it must be a url-safe base64-encoded string of 32 random bytes.
To generate a key with openssl:
```sh
openssl rand -base64 32
```

### Rotating the fields encryption key


The process of updating the database fields encryption `Secret` is manual because it requires manipulating sensitive data that the operator could not (or should not) have access (for example, if the `Secret` is stored in an external vault).

The `Secret` can contain multiple such keys (one per line). The key in the first line will be used for encryption but all others will still be attempted to decrypt old tokens. This can help you to rotate this key in the following way:

!!! WARNING
    Before proceeding, make sure to have a backup of Pulp database and the current fields encryption `Secret`.


* Shut down all Pulp services (api, content and worker pods).
```sh
$ PULP_CR=pulp
$ kubectl patch pulp $PULP_CR --type merge -p '{"spec": { "api": {"replicas":0},"content":{"replicas":0},"worker":{"replicas":0}}}'
```

* Add a new key at the top of the `Secret` key (modify the `NEW_SECRET` env var with your base64 encoded new encryption secret)
```sh
$ DB_ENCR_SECRET=$(kubectl get pulp $PULP_CR -ojsonpath='{.spec.db_fields_encryption_secret}')
$ OLD_SECRET=$(kubectl get secret $DB_ENCR_SECRET -ogo-template='{{index .data "database_fields.symmetric.key"}}')
$ NEW_SECRET=$(openssl rand -base64 32)
$ MERGE_SECRETS=$(printf "%s\n%s\n" "$NEW_SECRET" "$OLD_SECRET"|base64 -w0)
$ kubectl patch secret $DB_ENCR_SECRET --type merge -p "{\"data\": {\"database_fields.symmetric.key\": \"${MERGE_SECRETS}\"}}"
```

* Create a job to run `pulpcore-manager rotate-db-key`.
```yaml
$ kubectl apply -f-<<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: rotate-db-key
spec:
  template:
    spec:
      restartPolicy: "Never"
      containers:
      - name: pulpcore
        image: quay.io/pulp/pulp-minimal
        command: ["pulpcore-manager",  "rotate-db-key"]
        volumeMounts:
        - mountPath: /etc/pulp/settings.py
          name: pulp-server
          readOnly: true
          subPath: settings.py
        - mountPath: /etc/pulp/keys/database_fields.symmetric.key
          name: pulp-db-fields-encryption
          readOnly: true
          subPath: database_fields.symmetric.key
      volumes:
      - name: pulp-server
        secret:
          defaultMode: 420
          items:
          - key: settings.py
            path: settings.py
          secretName: pulp-server
      - name: pulp-db-fields-encryption
        secret:
          defaultMode: 420
          items:
          - key: database_fields.symmetric.key
            path: database_fields.symmetric.key
          secretName: pulp-db-fields-encryption
EOF
```

* Remove the old key (on the second line) from the `Secret`
```sh
$ MERGE_SECRETS=$(printf "%s\n" "$NEW_SECRET"|base64 -w0)
$ kubectl patch secret $DB_ENCR_SECRET --type merge -p "{\"data\": {\"database_fields.symmetric.key\": \"${MERGE_SECRETS}\"}}"
```

* Start the Pulp services again (make sure to adjust the number of replicas with the desired amount of pods)
```sh
$ kubectl patch pulp $PULP_CR --type merge -p '{"spec": { "api": {"replicas":1},"content":{"replicas":1},"worker":{"replicas":1}}}'
```
