# Pulp Operator storage configuration

Before installing Pulp, it is necessary to configure how Pulp should persist the data.

[Pulp uses django-storages](https://docs.pulpproject.org/pulpcore/installation/storage.html) to support multiple types of storage backends. The current version of pulp-operator supports the following types of storage installation:

* [Storage Class](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-pulp-operator-storage-to-use-a-storage-class)
* [Persistent Volume Claim](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-pulp-operator-storage-to-use-a-persistent-volume-claim)
* [Azure Blob](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-azure-blob-storage)
* [Amazon Simple Storage Service (S3)](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-aws-s3-storage)
* [Google Cloud Storage](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/storage/#configure-gcs-storage)

!!! info
    Only one storage type should be provided, trying to configure Pulp CR with multiple storage types will fail operator execution.


## Configure Pulp Operator storage to use a Storage Class

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


## Configure Pulp Operator storage to use a Persistent Volume Claim

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

## Configure Pulp Operator to use object storage

Pulp operator has the following parameters to configure Pulp core components with Object Storage:

* `ObjectStorageAzureSecret` - defines the name of the secret with Azure compliant object storage configuration.
* `ObjectStorageS3Secret` - defines the name of the secret with S3 compliant object storage configuration.
* `ObjectStorageGCSSecret` - defines the name of the secret with GCS compliant object storage configuration.

When Pulp operator is configured with one of the above parameters it is expected that the secrets are already present in the namespace of Pulp installation.
Pulp operator will automatically configure Pulp `settings.py` with the provided Object Storage backend.

!!! info
    Only one type of Object Storage should be provided. Trying to declare both will fail operator execution.

### Configure Azure Blob Storage

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

### Configure AWS S3 Storage

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

### Configure GCS Storage

Pulp Operator currently supports using GCS bucket using
[Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity).
This essentially means configuring the pods in the cluster to perform actions on
Google Cloud resources as a specified Service Account that has necessary
permissions. This means, only GKE is the supported k8s provider to use GCS as storage backend.

#### Prerequisites

* Enable necessary APIs

    ```
    # Set GCE project
    PROJECT_ID='foobar'
    gcloud config set project ${PROJECT_ID}

    # Enable required APIs
    gcloud services enable \
    container.googleapis.com \
    storage.googleapis.com \
    cloudresourcemanager.googleapis.com \
    iam.googleapis.com \
    iamcredentials.googleapis.com \
    compute.googleapis.com
    ```

* Create a [Service Account](https://docs.cloud.google.com/iam/docs/service-accounts-create) with the following properties:
    * Provide the following roles to the Service account
        * `roles/container.defaultNodeServiceAgent`
        * `roles/storage.objectAdmin` on the created GCS bucket
        * `roles/iam.serviceAccountTokenCreator`

        ```
        BUCKET_NAME='pulp-bucket'
        CLUSTER_NAME='pulp-cluster'
        GCE_SA='pulp-sa'
        GSA_EMAIL="${GCE_SA}@${PROJECT_ID}.iam.gserviceaccount.com"

        # Create the Google Service Account
        echo "Creating service account ${GCE_SA}"
        gcloud iam service-accounts create ${GCE_SA} \
            --display-name="Pulp Service Account" \
            --project=${PROJECT_ID} > /dev/null || echo "Service account ${GCS_SA} already exists"

        # Grant the Kubernetes Engine Default Node Service Agent role
        # This includes all necessary permissions for GKE nodes
        echo "Granting Kubernetes Engine Default Node SA role to ${GCE_SA}"
        gcloud projects add-iam-policy-binding ${PROJECT_ID} \
            --member="serviceAccount:${GSA_EMAIL}" \
            --role="roles/container.defaultNodeServiceAgent" \
            --condition=None > /dev/null

        # Allow necessary access to create signed URLs echo "Granting URL signing ability to ${GCE_SA}"
        echo "Allowing URL signing."
        gcloud iam service-accounts add-iam-policy-binding ${GSA_EMAIL} \
            --member="serviceAccount:${GSA_EMAIL}" \
            --role="roles/iam.serviceAccountTokenCreator" \
            --project=${PROJECT_ID} \
            --condition=None > /dev/null
        ```

    * Enable Workload identity by adding `roles/iam.workloadIdentityUser` role to
    `serviceAccount:${PROJECT_ID}.svc.id.goog[${PULP_NAMESPACE}/${PULP_SA}]`
    member. `PROJET_ID` is the GCP Project ID, `PULP_NAMESPACE` is the namespace
    in k8s where you deployed Pulp, and `PULP_SA` is the name of the Kubernetes
    Service Account used by Pulp (default is `pulp`)

        ```
        # Create IAM policy binding for Workload Identity
        echo "Create IAM policy binding for Workload Identity"
        gcloud iam service-accounts add-iam-policy-binding ${GSA_EMAIL} \
            --role roles/iam.workloadIdentityUser \
            --member "serviceAccount:${PROJECT_ID}.svc.id.goog[${PULP_NAMESPACE}/${PULP_SA}]" \
            --project=${PROJECT_ID} > /dev/null
        ```

* Create a [bucket](https://docs.cloud.google.com/storage/docs/creating-buckets) in a GCP Project to store the objects.

    ```
    # Create the GCS bucket and set permissions
    echo "Create GCS bucket ${BUCKET_NAME}"
    gcloud storage buckets create gs://${BUCKET_NAME} \
        --project=${PROJECT_ID} \
        --location=${REGION} \
        --uniform-bucket-level-access \
        --public-access-prevention  > /dev/null|| echo "Bucket already exists."

    # Allow access to the bucket to the SA
    echo "Giving access to bucket ${BUCKET_NAME}"
    gcloud storage buckets add-iam-policy-binding gs://${BUCKET_NAME} \
        --member="serviceAccount:${GSA_EMAIL}" \
        --role="roles/storage.objectAdmin" > /dev/null
    ```

* Tie Kubernetes Service Account created by Pulp to the Service Account created in GCP by adding `iam.gke.io/gcp-service-account: "${GSA_EMAIL}"` annotation.

    ```
    ...
    spec:
      sa_annotations:
        iam.gke.io/gcp-service-account: "${GSA_EMAIL}"
    ...
    ```

After performing all the prerequisites, create a `Secret` with them:

```
$ PULP_NAMESPACE='my-pulp-namespace'
$ GCS_BUCKET_NAME='my-aws-access-key-id'

$ kubectl -n $PULP_NAMESPACE apply -f- <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: 'test-gcs'
stringData:
  gcs-bucket-name: $GCS_BUCKET_NAME
  gcs-file-overwrite: True
  gcs-iam-sign-blob: True
  gcs-querystring-auth: True
EOF
```

Now configure `Pulp CR` with the secret created:
```
$ kubectl -n $PULP_NAMESPACE edit pulp
...
spec:
  object_storage_gcs_secret: test-gcs
...
```

After that, Pulp Operator will automatically update the `settings.py` config file and redeploy pulpcore pods to get the new configuration.
