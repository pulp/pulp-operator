# Scheduling Backups

The current version of Pulp Operator does not provide a way to automate the periodic execution of the backups. The following steps can be used as **an example** of how to create a [k8s Cronjob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) to schedule the backup execution.

* create a configmap with the `PulpBackup CR` definition (check the [backup section](/pulp_operator/backup_and_restore/config_running/#backup) for more information on `PulpBackup CR` fields configuration):
```yaml
$ kubectl apply -f- <<EOF
apiVersion: v1
data:
  pulp_backup.yaml: |
    apiVersion: repo-manager.pulpproject.org/v1beta2
    kind: PulpBackup
    metadata:
      name: pulpbackup
    spec:
      deployment_name: pulp
      backup_storage_class: standard
kind: ConfigMap
metadata:
  name: pulpbackup-cr
EOF
```

* for this example, we will create a new `ServiceAccount` that will be used by `Cronjob` pods. It is not a required step, you can skip it if your environment already has a `ServiceAccount` with the permissions to modify `PulpBackup` resources:
```yaml
$ kubectl apply -f-<<EOF
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pulpbackup
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pulpbackup
rules:
- apiGroups: ["repo-manager.pulpproject.org"]
  resources: ["pulpbackups"]
  verbs: ["get", "watch", "list","create","patch","update","delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pulpbackup
subjects:
- kind: ServiceAccount
  name: pulpbackup
roleRef:
  kind: Role
  name: pulpbackup
  apiGroup: rbac.authorization.k8s.io
EOF
```

* create the `k8s Cronjob` to run the backups:
```yaml
$ kubectl apply -f-<<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: pulpbackup
spec:
  schedule: "00 2 * * *"
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: bkp
            image: bitnami/kubectl:latest
            imagePullPolicy: IfNotPresent
            volumeMounts:
            - name: pulpbackup-cr
              mountPath: /tmp/pulp_backup.yaml
              subPath: pulp_backup.yaml
            command:
            - /bin/sh
            - -c
            args:
            - "kubectl apply -f /tmp/pulp_backup.yaml && kubectl wait --for condition=BackupComplete --timeout=600s -f /tmp/pulp_backup.yaml ; kubectl delete -f /tmp/pulp_backup.yaml"
          restartPolicy: Never
          serviceAccountName: pulpbackup
          volumes:
          - name: pulpbackup-cr
            configMap:
              name: pulpbackup-cr
EOF
```

In this example, the job:

* will be triggered every day at *2:00 AM* (`schedule: 00 2 * * *`)
* will keep `1` successful and/or `1` failed job (`successfulJobsHistoryLimit: 1, failedJobsHistoryLimit: 1`)
* will be considered failed if the backup does not finish in `10minutes` (`--timeout=600s`)
* will run with the previously created *pulpbackup* `ServiceAccount` (`serviceAccountName: pulpbackup`)


!!! note
    Don't forget to rotate the backup files from time to time to avoid filling up the storage.