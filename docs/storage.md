# Pulp Operator storage configuration

Before installing Pulp, for production clusters, it is necessary to configure how Pulp should persist the data.

Pulp uses `django-storages` to support multiple types of storage backends. The current version of operator supports the following types of storage installation:

* [Storage Class](#configuring-pulp-operator-storage-to-use-a-storage-class)
* [Persistent Volume Claim](#configuring-pulp-operator-storage-to-use-a-persistent-volume-claim)
* [Azure Blob](#configuring-pulp-operator-to-use-object-storage)
* [Amazon Simple Storage Service (S3)](#configuring-pulp-operator-to-use-object-storage)
* [EmptyDir](#configuring-pulp-operator-in-non-production-clusters)

!!! info
    Only one storage type should be provided, trying to configure Pulp CR with multiple storage types will fail operator execution.


If no backend is configured, Pulp will by default use the EmptyDir volume.


## Configuring Pulp Operator storage to use a Storage Class

Pulp operator has the following parameters to configure the components with a Storage Class:

* `FileStorageClass` - defines the name of the Storage Class that will be used by Pulp core pods
* `Database.PostgresStorageClass` - defines the name of the Storage Class that will be used by Database pods
* `Cache.RedisStorageClass` - defines the name of the Storage Class that will be used by Cache pods

When Pulp operator is configured with the above parameters it will automatically provision new Persistent Volume Claims with the Storage Class provided.

To verify if there is a Storage Class available:
```
$ kubectl get sc
```

If the Kubernetes cluster has no Storage Class configured, it is possible to configure Pulp with other parameters of storage or follow the [steps to create a new Storage Class](https://kubernetes.io/docs/concepts/storage/storage-classes/).



## Configuring Pulp Operator storage to use a Persistent Volume Claim

Pulp operator has the following parameters to configure the components with a Persistent Volume Claim:

* `PVC` - defines the name of the Persistent Volume Claim that will be used by Pulp core pods
* `Database.PVC` - defines the name of the Persistent Volume Claim that will be used by Database pods
* `Cache.PVC` - defines the name of the Persistent Volume Claim that will be used by Cache pods

When Pulp operator is configured with the above parameters it is expected that the PVCs are already provisioned and Pulp operator will automatically configure the Deployments and StatefulSet with them.

To verify the list of Persistent Volume Claims available:
```
$ kubectl get pvc
```

If the installation namespace has no Persistent Volume Claim available, it is possible to configure Pulp with other parameters of storage or follow the [steps to create a new Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims).



## Configuring Pulp Operator to use object storage

Pulp operator has the following parameters to configure Pulp core components with Object Storage:

* `ObjectStorageAzureSecret` - defines the name of the secret with Azure compliant object storage configuration. 
* `ObjectStorageS3Secret` - defines the name of the secret with S3 compliant object storage configuration.

When Pulp operator is configured with one of the above parameters it is expected that the secrets are already present in the namespace of Pulp installation.
Pulp operator will automatically configure Pulp `settings.py` with the provided Object Storage backend.

!!! info
    Only one type of Object Storage should be provided. Trying to declare both will fail operator execution.


## Configuring Pulp Operator in non-production clusters

If there is no `Storage Class` nor `Persistent Volume Claim` nor `Object Storage` provided the operator will deploy the components (Pulp, Database, and Cache) with an [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir).

You must configure storage for the Pulp Operator. For non-production clusters, you can set the components to an empty directory. If you do so, everything is lost if you restart the pod.

!!! warning
    Configure this option for only non-production clusters.

