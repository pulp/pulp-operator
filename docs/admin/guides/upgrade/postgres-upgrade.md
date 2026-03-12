# PostgreSQL Major Version Upgrade Steps

The following steps will guide you through a major version upgrade of PostgreSQL (e.g., 13 to 15).
A shutdown of all Pulp pods is required, so a maintenance window is recommended.


> **ℹ️ NOTE**
>
> **The steps were tested only with postgres deployed by the operator. For steps to upgrade external
> databases follow your vendor documentation.**

## Prerequisites

* make sure you have enough storage space on the machine running `kubectl` (we will copy the dump locally)
* **make sure to have a backup of the environment**

To get an approximate size of the database:
```sh
kubectl -n $PULP_NAMESPACE exec ${PULP_CR}-database-0 -- psql -U pulp -c "SELECT pg_size_pretty(pg_database_size('pulp'));"
```

## Step 1 — Set env vars

* these environment variables will be used several times in the next steps
```sh
PULP_CR=pulp
PULP_NAMESPACE=pulp
```

## Step 2 — Scale Down Pulp Components

Scale down all Pulp pods to avoid any writes to the database:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"api":{"replicas":0},"content":{"replicas":0},"worker":{"replicas":0}}}'
```

* if you have pulp web running:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"web":{"replicas":0}}}'
```

**Wait for all API, content, and worker pods to terminate:**
```sh
kubectl -n $PULP_NAMESPACE get pods -w
```

## Step 3 — Dump the Database

Run `pg_dump` inside the running database pod and store it in `/tmp/pulp.db` on the local machine:
```sh
kubectl -n $PULP_NAMESPACE exec ${PULP_CR}-database-0 -- pg_dump --clean -Ft -U pulp -d pulp > /tmp/pulp.db
```

## Step 4 — Delete the Old Database StatefulSet and PVC

Put the operator in unmanaged state to avoid the StatefulSet redeploy:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"unmanaged":true}}'
```

Delete the postgres StatefulSet and PVC:
```sh
kubectl -n $PULP_NAMESPACE delete statefulset ${PULP_CR}-database
kubectl -n $PULP_NAMESPACE delete pvc ${PULP_CR}-postgres-${PULP_CR}-database-0
```

## Step 5 — Update the Pulp CR to Use the New PostgreSQL Version

Update the `postgres_image` to the desired version (e.g., PostgreSQL 15):
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"database":{"postgres_image":"docker.io/library/postgres:15"}}}'
```

Put the operator back to a managed state:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"unmanaged":false}}'
```

**Wait for the operator to reconcile and the new database pod to be running:**
```sh
kubectl -n $PULP_NAMESPACE wait --for condition=Pulp-Operator-Finished-Execution pulp/$PULP_CR --timeout=900s
```

```sh
kubectl -n $PULP_NAMESPACE get pods -w
```

## Step 6 — Restore the Dump into the New PostgreSQL

Copy the dump to the new database pod:
```sh
kubectl -n $PULP_NAMESPACE cp /tmp/pulp.db ${PULP_CR}-database-0:/tmp/pulp.db
```

Run the `pg_restore`:
```sh
kubectl -n $PULP_NAMESPACE exec ${PULP_CR}-database-0 -- bash -c 'pg_restore --clean --if-exists -U pulp -d pulp /tmp/pulp.db'
```

If `pg_restore` reports errors about objects already existing (e.g., the `pulp` role), that is
expected since `--clean` drops before creating but some default objects may already exist.
Verify the exit code (`echo $?`) — a non-zero exit due to pre-existing objects is harmless,
but data-related errors should be investigated.

Clean up the dump file from the database pod:
```sh
kubectl -n $PULP_NAMESPACE exec ${PULP_CR}-database-0 -- rm /tmp/pulp.db
```

## Step 7 — Scale Pulp Back Up

Make sure to set the number of replicas according to your environment needs:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"api":{"replicas":1},"content":{"replicas":1},"worker":{"replicas":1}}}'
```

* if you have pulp web running:
```sh
kubectl -n $PULP_NAMESPACE patch pulp $PULP_CR --type merge -p '{"spec":{"web":{"replicas":1}}}'
```

Wait for the pods to get into a running state:
```sh
kubectl -n $PULP_NAMESPACE wait --for condition=Pulp-Operator-Finished-Execution pulp/$PULP_CR --timeout=900s
```

## Step 8 — Verify

Confirm PostgreSQL version
```sh
kubectl -n $PULP_NAMESPACE exec ${PULP_CR}-database-0 -- psql -U pulp -c "SELECT version();"
```

Check pods are healthy
```sh
kubectl -n $PULP_NAMESPACE get pods
```

Check pulp status (requires [Pulp CLI](https://docs.pulpproject.org/pulp_cli/) installed and configured):
```sh
pulp status
```

You can also clean up the local dump file once you have confirmed the upgrade is successful:
```sh
rm /tmp/pulp.db
```

## Rollback

If the restore fails, the local dump file (`/tmp/pulp.db`) can be used to retry.
Repeat from Step 4 to re-create the database pod and restore again.
