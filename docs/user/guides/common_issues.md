### **I modified a configmap/secret but the new config is not replicated to pods**

This is a known issue that [Pulp team is discussing](https://github.com/pulp/pulp-operator/issues/521) what will be the best way to handle it. The [Kubernetes community does not have a consensus](https://github.com/kubernetes/kubernetes/issues/22368) about it too.

One of the reasons that we don't want to put the pod restart logic (to propagate the new configuration) in operator is because it can cause downtime in case of error and we would need to automate the rollback or fix processes, which would probably bring other issues instead of being helpful.



### **How can I fix the "*Multi-Attach error for volume "my-volume" Volume is already used by pod(s) my-pod*"?**

    Warning  FailedAttachVolume  44m                   attachdetach-controller  Multi-Attach error for volume "my-volume" Volume is already used by pod(s) my-pod

The `Multi-Attach` error happens when the deployment is configured to mount a RWO volume and the scheduler tries to run a new pod in a different node. To fix the error and allow the new pod to be provisioned, delete the old pod so that the new one will be able to mount the volume.

Another common scenario that this issue happens is if a node lose communication with the cluster and the controller tries to provision a new pod. Since the communication of the failing node with the cluster is lost, it is not possible to detach the volume before trying to attach it again to the new pod.

To avoid this issue, it is recommended to use RWX volumes. If it is not possible, modify the `strategy` type from Pulp CR as `Recreate`.


### **Which resources are managed by the Operator?**

The Operator manages almost all resources it provisions, which are:

* **Deployments**
    * `pulp-api`, `pulp-content`, `pulp-worker`, *`pulp-web`(optional)*, *`pulp-redis`(optional)*
* **Services**
    * `pulp-api-svc`, `pulp-content-svc`, *`pulp-web-svc`(optional)*,*`pulp-redis-svc`(optional)*
* ***Routes(optional)***
    * *pulp, pulp-content, pulp-api, pulp-auth, pulp-container-v2, ...*
* ***Ingresses(optional)***
    * *pulp*



Some resources are provisioned by the Operator, but [**no reconciliation**](/pulp_operator/configuring/secrets/#pulp-operator-secrets) is made (even in managed state):

* **ConfigMaps**
* **Secrets**


!!! note
    Keep in mind that this list is constantly changing (sometimes we are adding more resources,
    sometimes we identify that a resource is not needed anymore).


### **Which fields are reconciled when the Operator is set as `managed`?**

All fields from `Spec` *should* be reconciled on *Deployments*, *Services*, *Routes* and *Ingresses* objects.


### **I created a new PulpRestore CR, but the restore procedure is not running again. Checking the operator logs I found the message "*PulpRestore lock ConfigMap found. No restore procedure will be executed!*"**

After the Operator finishes executing a restore procedure it creates a `ConfigMap` called *`restore-lock`*. This `ConfigMap` is used to control the restore reconciliation loop and avoid it overwriting all files or `Secrets` with data from an old backup.  
If you still want to run the restore, just delete the *`restore-lock`* `ConfigMap` and recreate the PulpRestore CR.


### **How can I manually run a database migration?**

There are some cases in which a db migration is needed and the operator did not automatically trigger a new migration job.  
We are investigating how the operator should handle these scenarios.

In the meantime, to manually run a migration if there is/are pulpcore pods running:
```bash
$ kubectl exec deployments/<PULP CR NAME>-api pulpcore-manager migrate
```

for example:
```bash
$ kubectl exec deployments/pulp-api pulpcore-manager migrate
```

If the pods are stuck in init or crash state and Pulp is running in an OCP cluster:
```bash
$ oc debug -c init-container  deployment/pulp-api -- pulpcore-manager migrate
```

or create a job to run the migrations (the values from these vars can be extracted from pulpcore pods):
```bash
export POSTGRES_HOST="pulp-database-service"
export POSTGRES_PORT="5432"
export SERVICE_ACCOUNT="pulp"
export PULP_SERVER_SECRET="pulp-server"
export DB_FIELDS_ENC_SECRET="pulp-db-fields-encryption"


kubectl apply -f-<<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: pulpcore-migration
spec:
  backoffLimit: 1
  completionMode: NonIndexed
  completions: 1
  parallelism: 1
  template:
    spec:
      containers:
      - args:
        - -c
        - |-
          /usr/bin/wait_on_postgres.py
          /usr/local/bin/pulpcore-manager migrate --noinput
        command:
        - /bin/sh
        env:
        - name: POSTGRES_SERVICE_HOST
          value: "$POSTGRES_HOST"
        - name: POSTGRES_SERVICE_PORT
          value: "$POSTGRES_PORT"
        image: quay.io/pulp/pulp-minimal:stable
        name: migration
        volumeMounts:
        - mountPath: /etc/pulp/keys/database_fields.symmetric.key
          name: pulp-db-fields-encryption
          readOnly: true
          subPath: database_fields.symmetric.key
        - mountPath: /etc/pulp/settings.py
          name: pulp-server
          readOnly: true
          subPath: settings.py
      restartPolicy: Never
      serviceAccount: "$SERVICE_ACCOUNT"
      volumes:
      - name: pulp-server
        secret:
          defaultMode: 420
          items:
          - key: settings.py
            path: settings.py
          secretName: "$PULP_SERVER_SECRET"
      - name: pulp-db-fields-encryption
        secret:
          defaultMode: 420
          items:
          - key: database_fields.symmetric.key
            path: database_fields.symmetric.key
          secretName: "$DB_FIELDS_ENC_SECRET"
EOF
```
