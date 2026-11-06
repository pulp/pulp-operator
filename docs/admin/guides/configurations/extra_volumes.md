# Configure Extra Volumes and Service Accounts

Pulp Operator provides the `extra_volumes`, `extra_volume_mounts` and `service_account_name` fields to customize the pods of each pulpcore component (`api`, `content` and `worker`). <br/>

!!! INFO
    * `extra_volume_mounts` are added to the main container only.
    * Jobs and the `web` pods do not use these fields.
    * The ServiceAccount defined in `service_account_name` must already exist. It is not created by Pulp Operator.
    * Pods using a custom ServiceAccount get `image_pull_secrets` set in the pod spec.

For more information about Kubernetes volumes and service accounts, check the k8s official documentation:

* [Volumes](https://kubernetes.io/docs/concepts/storage/volumes/)
* [Configure Service Accounts for Pods](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)


## Worker Deployments

Example of CR configuration for `pulpcore-worker` pods:
```yaml
spec:
  worker:
    service_account_name: pulp-worker
    extra_volumes:
    - name: signing-scripts
      secret:
        secretName: my-signing-scripts
        defaultMode: 0755
    extra_volume_mounts:
    - name: signing-scripts
      mountPath: /opt/pulp-scripts
```

The same fields are available for `api` and `content`:
```yaml
spec:
  api:
    extra_volumes:
    - name: secrets-store
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: my-secrets
    extra_volume_mounts:
    - name: secrets-store
      mountPath: /mnt/secrets
      readOnly: true
```
