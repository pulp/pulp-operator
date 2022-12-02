# Galaxy

Pulp Operator can also be used to deploy [Galaxy](https://galaxyng.netlify.app/), a Pulp plugin to support hosting your very own Ansible Galaxy server.

## Deploy and Sync default EE images

It is possible to configure the operator to automatically deploy a set of *Execution Environments*.
It does so by periodic running a [*skopeo sync*](https://github.com/containers/skopeo/blob/main/docs/skopeo-sync.1.md#name) command through a k8s [`CronJob`](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/).

To enable this feature, update Pulp CR field `deploy_ee_defaults`:
```
...
spec:
  deploy_ee_defaults: true
...
```

By default, if it is not defined, it will be considered *false*.


!!! Info
    List of default images synchronized:

    * quay.io/fedora/fedora
    * quay.io/fedora/fedora-minimal

### Configuring the list of images to be synced

If not provided, the operator will create a `ConfigMap` called `ee-default-images` with a custom list of *Execution Environments* to be synchronized.

During the installation, it is also possible to define the name of a custom `ConfigMap` using the `ee_defaults` field:
```
...
spec:
  deploy_ee_defaults: true
  ee_defaults: <name of the ConfigMap with the list of EE>
...
```

Here is an example of how to create the `ConfigMap`:
```
$ kubectl apply -f-<<EOF
apiVersion: v1
data:
  images.yaml: |-
    quay.io:
      images-by-tag-regex:
        fedora/fedora-minimal: ^latest$
        fedora/fedora: ^latest$
kind: ConfigMap
metadata:
  name: <name of the ConfigMap with the list of EE>-
EOF
```

The `ConfigMap` must have the following structure:

* a key named `images.yaml`
* a [yaml content](https://github.com/containers/skopeo/blob/main/docs/skopeo-sync.1.md#yaml-file-content-used-source-for---src-yaml) with the list of images to be copied

Check [skopeo repo doc](https://github.com/containers/skopeo/blob/main/docs/skopeo-sync.1.md#yaml-file-content-used-source-for---src-yaml) for more information on the YAML file content format.


### Configuring the CronJob resource

When `deploy_ee_defaults` is set true, a `CronJob` resource will be created to schedule `Jobs` that will provision `Pods` to run the *`skopeo sync`* command.  
By default, the sync is scheduled to run every two minutes, but it can be changed through the `.spec.schedule` field from `CronJob`:
```
apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-test-cronjob
spec:
  schedule: "*/2 * * * *"
```

See Kubernetes `CronJob` documentation for more information of the fields available: [https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/).

!!! Note
    The current version of Pulp Operator is **not** reconciling the `ConfigMap` and `CronJob`
    used to sync the images. So, after deploying these resource you can modify it directly and
    the operator will not try to rollback the changes. If you wish to discard any change and use
    the default values, just delete the `CronJob` and the operator will re-provision a new one.