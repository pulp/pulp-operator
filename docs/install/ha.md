# HIGHLY AVAILABLE PULP


Pulp’s architecture is designed to offer scalability and high availability. You can scale Pulp’s architecture in whatever way suits your needs.

With Pulp, the more you increase your availability, you also increase your capability. The more `Pulpcore API` replicas you deploy, the more API requests you can serve. The more `Pulpcore content` replicas you deploy, the more binary data requests you can serve. The more `Pulpcore workers` you start, the higher the tasking (syncing, publishing) throughput you deployment can handle.

You are free to, for example, deploy all of the components onto separate nodes, deploy multiple `Pulpcore API`, `content` serving application, `Pulpcore worker` services on a single node, as long as they can communicate with one another, there are no limits.


## PRE-REQS

Any service which Pulp Operator **does not** deploy in a highly available deployment, is an external service.
The following services are considered external:

* Database
* Cache
* Shared/Object storage

### DATABASE

For high availability deployments, you will need to configure a clustered PostgreSQL. There are very good operators to deploy a clustered PostgreSQL and cloud databases (like AWS RDS).
After deploying the database follow the steps from [Configuring Pulp Operator to use an external PostgreSQL installation](https://docs.pulpproject.org/pulp_operator/configuring/database/#configuring-pulp-operator-to-use-an-external-postgresql-installation) to set Pulp CR with it.

### CACHE

Redis helps to increase the speed at which the content app caches information about requests. This cache makes answering subsequent content-app requests easier. However, it’s optional, you are free not to use it. If it fails, Pulp will continue to work.
After deploying a clustered Redis follow the steps from [Configuring Pulp Operator to use an external Redis installation](https://docs.pulpproject.org/pulp_operator/configuring/cache/#configuring-pulp-operator-to-use-an-external-redis-installation) to set Pulp CR with it.

### STORAGE BACKEND

Pulp requires a storage backend, such as a filesystem like NFS, Samba or cloud storage, to read and write data that is then shared between Pulpcore processes. Follow the steps from [Pulp Operator storage configuration](https://docs.pulpproject.org/pulp_operator/configuring/storage/) to configure Pulp CR with the storage backend.


## SCALING PULPCORE PODS

It is possible to modify the number of `Pulpcore API`, `content`, and `worker` Pod replicas to, for example, spread the load among different Pods or avoid disruption in case of a Pod failure.
To change the number of replicas of each component, update the `.spec.<component>.replicas` field(s) from Pulp CR:
```yaml
spec:
  api:
    replicas: 3
  content:
    replicas: 3
  worker:
    replicas: 3
```

## AFFINITY RULES

Pulp Operator can define a group of affinity scheduling rules. With affinity rules it is possible to set constrains like in which node a pod should run (like in `nodeSelectors`) or inter pod affinity/anti-affinity to define if Pods should/should not run in the same node that another Pod with a defined label is running.  
To configure an affinity rule, update the `spec.<component>.affinity` field(s) from Pulp CR. In the following example we are configuring Pulp Operator to avoid deploying the same `Pulpcore` component in nodes that have different values of `topology.kubernetes.io/zone` label (`pulp-api-1` would run in a node with label `topology.kubernetes.io/zone: us-east-1a`, `pulp-api-2` in `topology.kubernetes.io/zone: us-east-1b`, and `pulp-api-3` in `topology.kubernetes.io/zone: us-east-1c`):
```yaml
spec:
  api:
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/component
                operator: In
                values:
                - api
            topologyKey: topology.kubernetes.io/zone
  content:
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/component
                operator: In
                values:
                - content
            topologyKey: topology.kubernetes.io/zone
  worker:
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/component
                operator: In
                values:
                - worker
            topologyKey: topology.kubernetes.io/zone
```

!!! note
    The `topologyKey` should match the labels from the nodes in which the pods should be co-located(affinity) or **not** co-located(anti-affinity).



* Check the official k8s documentation for more information about kubernetes affinity rules: [https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node).
* For more information on how to configure affinity rules in Pulp Operator check: [https://docs.pulpproject.org/pulp_operator/configuring/podPlacement/](https://docs.pulpproject.org/pulp_operator/configuring/podPlacement/)



## PDB (Pod Disruption Budget)

"*Kubernetes offers features to help you run highly available applications even when you introduce frequent voluntary disruptions.
As an application owner, you can create a PodDisruptionBudget (PDB) for each application. A PDB limits the number of Pods of a replicated application that are down simultaneously from voluntary disruptions.*" [#pod-disruption-budgets](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets)

Suppose that you are in the middle of a critical task (repo sync, upgrade, migration, etc) and you don’t want it to be interrupted, meaning, no running `API` or `content` pod should be deleted even if a cluster admin tries to put a node in maintenance mode or drain the node. For this we can use PDB:
```yaml
spec:
  api:
    replicas: 3
    pdb:
      minAvailable: 3
  content:
    replicas: 3
    pdb:
      maxUnavailable: 0
  worker:
    replicas: 2
    minAvailable: 1
```

!!! note
    You don't need to specify the `PDB` pod label selector, the operator will add it automatically based on the `spec.{api,worker,content}` field.

!!! warning
    Make sure you know what you are doing before configuring PDB. If not configured correctly it can cause unexpected behavior, like getting a node in a hang state during maintenance (node drain) or cluster upgrade.

* Check the official k8s documentation for more information about kubernetes PDB: [https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/#pod-disruption-budgets).
* For more information on how to configure PDB in Pulp Operator check: [https://docs.pulpproject.org/pulp_operator/configuring/pdb/](https://docs.pulpproject.org/pulp_operator/configuring/pdb/)

## ROLLING UPDATE DEPLOYMENT STRATEGY

Suppose that a new image of `Pulpcore` is released and you want to use it. To update the `Pulpcore` image version deployed by Pulp Operator (`.spec.image_version` field) with no downtime, it is possible to set the `Deployment Strategy` as `RollingUpdate` (default value if none provided).  

For example, if we set the `.spec.strategy` to `RollingUpdate` and `maxUnavailable` to 50%, if we change the `.spec.image_version` from *3.32* to *3.33*, kubernetes will keep half of the replicas running and reprovision the other half with the new image. When the new images get into a READY state it will start to redeploy the older pods, executing the upgrade with no downtime.
Example of configuration:
```yaml
spec:
  api:
    strategy:
      type: RollingUpdate
      rollingUpdate:
        maxUnavailable: 50%
  content:
    strategy:
      type: RollingUpdate
      rollingUpdate:
        maxUnavailable: 50%
```



!!! warning
    If the `Deployment.Strategy` is set to `Recreate`, kubernetes will ensure that all pods are `Terminated` before starting the new replicas, which will cause downtime during a deployment rollout.


* Check the official k8s documentation for more information about `Deployment Strategy`: [https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy).

## NODE SELECTOR

The `Pulp workers` are part of the tasking system. Because syncing large amounts of content takes time, a lot of the work in Pulp runs longer than is suitable for webservers. You can deploy as many workers as you need. Since we can scale up/down `Pulpcore Worker` pods on demand, it is possible to configure node selector to deploy worker pods in spot instances, for example:

```yaml
# is_spot_instance is a node label that you should manually set to the expected nodes
spec:
  worker:
    node_selector:
      is_spot_instance: "true"
```

Anothe example of usage of `nodeSelectors` is to schedule Pulp pods in specific nodes, like run `API` pods on nodes with `beta.kubernetes.io/instance-type=t2.medium` label and `content` pods on `beta.kubernetes.io/instance-type=c7g.2xlarge`.

* Check the official k8s documentation for more information on `node labels` and `nodeSelector`: [https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#built-in-node-labels](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#built-in-node-labels)
* For more information on how to configure `nodeSelector` in Pulp Operator check: [https://docs.pulpproject.org/pulp_operator/configuring/podPlacement/](https://docs.pulpproject.org/pulp_operator/configuring/podPlacement/)

## SINGLE-SHOT SCRIPT

The following script consolidates the HA features present in this doc and will deploy the following components:

* a `Secret` with the information to connect to a running PostgreSQL
* a `Secret` with the object storage (S3) credentials
* a `Secret` with the information to connect to a running Redis 
* an example of `Pulp CR` to deploy a highly available instance of Pulp

!!! warning
    This script should be used as an example only!

```sh
#!/bin/bash

kubectl config set-context --current --namespace pulp

kubectl apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
  name: external-database
stringData:
  POSTGRES_HOST: my-clustered-postgres-host.example.com
  POSTGRES_PORT: '5432'
  POSTGRES_USERNAME: 'pulp'
  POSTGRES_PASSWORD: 'password'
  POSTGRES_DB_NAME: 'pulp'
  POSTGRES_SSLMODE: 'prefer'
EOF

kubectl apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pulp-object-storage
stringData:
  s3-access-key-id: my-object-storage-key-id
  s3-secret-access-key: my-object-storage-access-key
  s3-bucket-name: 'pulp3'
  s3-region: us-east-1
EOF

kubectl apply -f-<<EOF
apiVersion: v1
kind: Secret
metadata:
  name: external-redis
stringData:
  REDIS_HOST: my-redis-host.example.com
  REDIS_PORT: '6379'
  REDIS_PASSWORD: ""
  REDIS_DB: ""
EOF

kubectl apply -f-<<EOF
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: test-pulp-ha
spec:
  object_storage_s3_secret: pulp-object-storage
  database:
    external_db_secret: external-database
  cache:
    enabled: true
    external_cache_secret: external-redis
  api:
    replicas: 6
    strategy:
      rollingUpdate:
        maxUnavailable: 30%
      type: RollingUpdate
    pdb:
      minAvailable: 3
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/component
                operator: In
                values:
                - api
            topologyKey: topology.kubernetes.io/zone
  content:
    replicas: 6
    pdb:
      maxUnavailable: 50%
    strategy:
      rollingUpdate:
        maxUnavailable: 30%
      type: RollingUpdate
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
              - key: app.kubernetes.io/component
                operator: In
                values:
                - content
            topologyKey: topology.kubernetes.io/zone
  worker:
    pdb:
      minAvailable: 2
    replicas: 6
    node_selector:
      is_spot_instance: "true"
  web:
    replicas: 1

  file_storage_access_mode: "ReadWriteOnce"
  file_storage_size: "2Gi"
  file_storage_storage_class: standard
EOF
```