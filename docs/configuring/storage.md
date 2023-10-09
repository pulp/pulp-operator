# Pulp Operator storage configuration

Before installing Pulp, for production clusters, it is necessary to configure how Pulp should persist the data.

[Pulp uses django-storages](https://docs.pulpproject.org/pulpcore/installation/storage.html) to support multiple types of storage backends. The current version of operator supports the following types of storage installation:

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


!!! note
    If the Storage Class defined will provision RWO volumes, it is recommended to also set the [`Deployment strategy`](https://docs.pulpproject.org/pulp_operator/pulp/) in Pulp CR as [`Recreate`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#recreate-deployment) to avoid the [`Multi-Attach`](https://docs.pulpproject.org/pulp_operator/faq/#how-can-i-fix-the-multi-attach-error-for-volume-my-volume-volume-is-already-used-by-pods-my-pod) volume error.

Here is an example to deploy Pulp in a persistent way providing different `StorageClasses` for each component:
```
spec:
  file_storage_storage_class: my-sc-for-pulpcore
  file_storage_size: "10Gi"
  file_storage_access_mode: "ReadWriteMany"
  database:
    postgres_storage_class: my-sc-for-database
  cache:
    redis_storage_class: my-sc-for-cache
```


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


!!! note
    If the Persistent Volume Claim defined is bound to a RWO volume, it is recommended to also set the [`Deployment strategy`](https://docs.pulpproject.org/pulp_operator/pulp/) in Pulp CR as [`Recreate`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#recreate-deployment) to avoid the [`Multi-Attach`](https://docs.pulpproject.org/pulp_operator/faq/#how-can-i-fix-the-multi-attach-error-for-volume-my-volume-volume-is-already-used-by-pods-my-pod) volume error.

Here is an example to deploy Pulp in a persistent way providing different `PersistentVolumeClaims` for each component:
```
spec:
  pvc: my-pvc-for-pulpcore
  database:
    pvc: my-pvc-for-database
  cache:
    pvc: my-pvc-for-cache
```

## Configuring Pulp Operator to use object storage

Pulp operator has the following parameters to configure Pulp core components with Object Storage:

* `ObjectStorageAzureSecret` - defines the name of the secret with Azure compliant object storage configuration.
* `ObjectStorageS3Secret` - defines the name of the secret with S3 compliant object storage configuration.

When Pulp operator is configured with one of the above parameters it is expected that the secrets are already present in the namespace of Pulp installation.
Pulp operator will automatically configure Pulp `settings.py` with the provided Object Storage backend.

!!! info
    Only one type of Object Storage should be provided. Trying to declare both will fail operator execution.

### Configuring Azure Blob

#### Prerequisites
* To configure Pulp with Azure Blob as a storage backend, the first thing to do is create an [Azure Storage Blob Container](https://docs.microsoft.com/en-us/azure/storage/blobs/quickstart-storage-explorer) to store the objects.
* After configuring a `Blob Container`, take a note of the [Azure storage account](https://docs.microsoft.com/en-us/azure/storage/common/storage-account-get-info?toc=%2Fazure%2Fstorage%2Fblobs%2Ftoc.json&tabs=portal)

After performing all the prerequisites, create a `Secret` with them:
```
$ PULP_NAMESPACE='my-pulp-namespace'
$ AZURE_ACCOUNT_NAME='my-azure-account-name'
$ AZURE_ACCOUNT_KEY='my-azure-account-key'
$ AZURE_CONTAINER='pulp-test'
$ AZURE_CONTAINER_PATH='pulp3'
$ AZURE_CONNECTION_STRING='my-azure-connection-string'

$ kubectl -n $PULP_NAMESPACE apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: 'test-azure'
stringData:
  azure-account-name: $AZURE_ACCOUNT_NAME
  azure-account-key: $AZURE_ACCOUNT_KEY
  azure-container: $AZURE_CONTAINER
  azure-container-path: $AZURE_CONTAINER_PATH
  azure-connection-string: $AZURE_CONNECTION_STRING
EOF
```

!!! note
    `azure-connection-string` is an **optional** field that can be used to keep compatibility with other Azure Storage compliant systems, like [Azurite](https://github.com/Azure/Azurite).

Now configure `Pulp CR` with the secret created:
```
$ kubectl -n $PULP_NAMESPACE edit pulp
...
spec:
  object_storage_azure_secret: test-azure
...
```

After that, Pulp Operator will automatically update the `settings.py` config file and redeploy pulpcore pods to get the new configuration.

### Configure AWS S3

#### Prerequisites
* To configure Pulp with AWS S3 as a storage backend, the first thing to do is create a [S3 Bucket](https://docs.aws.amazon.com/AmazonS3/latest/userguide/creating-bucket.html) to store the objects.
* After configuring a `S3 Bucket` take a note of the [AWS credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html)

After performing all the prerequisites, create a `Secret` with them:
```
$ PULP_NAMESPACE='my-pulp-namespace'
$ S3_ACCESS_KEY_ID='my-aws-access-key-id'
$ S3_SECRET_ACCESS_KEY='my-aws-secret-access-key'
$ S3_BUCKET_NAME='pulp3'
$ S3_REGION='us-east-1'

$ kubectl -n $PULP_NAMESPACE apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: 'test-s3'
stringData:
  s3-access-key-id: $S3_ACCESS_KEY_ID
  s3-secret-access-key: $S3_SECRET_ACCESS_KEY
  s3-bucket-name: $S3_BUCKET_NAME
  s3-region: $S3_REGION
EOF
```
If you want to use a custom S3-compatible endpoint, you can use it by specifying the endpoint 
within the secret data as `s3-endpoint`.
In this case `s3-region` does not need to be specified and is ignored.

Now configure `Pulp CR` with the secret created:
```
$ kubectl -n $PULP_NAMESPACE edit pulp
...
spec:
  object_storage_s3_secret: test-s3
...
```

After that, Pulp Operator will automatically update the `settings.py` config file and redeploy pulpcore pods to get the new configuration.


## Configuring Pulp Operator in non-production clusters

If there is no `Storage Class` nor `Persistent Volume Claim` nor `Object Storage` provided the operator will deploy the components (Pulp, Database, and Cache) with an [emptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir).

You must configure storage for the Pulp Operator. For non-production clusters, you can set the components to an empty directory. If you do so, everything is lost if you restart the pod.

!!! warning
    Configure this option for only non-production clusters.

!!! warning
    The content stored in an `emptyDir` volume is not shared between the pods, because of that deploying Pulp with more than a single replica of `pulpcore-api` and/or `pulpcore-content` will result in unexpected behaviors.

Configuring Pulp operator with `emptyDir` will fail the execution of some plugins.
For example, `pulp-container` plugin needs to access the data created by `pulpcore-api` component through `pulpcore-content` pod, but since each pod has its own `emptyDir` volume - and their data is not shared between them - Pulp will not work as expected.

