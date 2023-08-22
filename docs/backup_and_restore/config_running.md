# Configuring and Running


All the configurations needed to run the backup or restore procedures are made through `PulpBackup` or `PulpRestore` CRs.

To get the list of fields available in each CR, check [`PulpBackupSpec`](/pulp_operator/backup/#pulpbackupspec) and [`PulpRestoreSpec`](/pulp_operator/restore/#pulprestorespec).

## Backup

To configure the backup controller, create a manifest file with the definition of [PulpBackup CR](/pulp_operator/backup/#pulpbackupspec).  
For example:
```
---
apiVersion: repo-manager.pulpproject.org/v1beta2
kind: PulpBackup
metadata:
  name: pulpbackup-sample
spec:
  deployment_name: pulp
  deployment_type: pulp
  backup_storage_class: standard
  admin_password_secret: example-pulp-admin-password
  postgres_configuration_secret: pulp-postgres-configuration
```

In the above sample we defined:

* the name of `Pulp` instance (`deployment_name`), which can be gathered with:
```
$ kubectl get pulp
NAME   AGE
pulp   3m30s
```

* the type of deployment (`deployment_type`), in this case `pulp` (but could also be `galaxy` depending on the installation).
* the name of `StorageClass` used to provision the `PVC` to store the backup data (`backup_storage_class`).
* the name of the `Secret` with Pulp admin password (`admin_password_secret`), which can be get by:
```
$ kubectl get pulp pulp -ojsonpath='{.spec.admin_password_secret}{"\n"}'
pulp-admin-password
```

* the name of the `Secret` with PostgreSQL credentials and connection information (`postgres_configuration_secret`).

After finishing to configure the file, apply the configuration and the Operator will start the backup:
```
kubectl apply -f <backup_cr_file>.yaml
```


## Restore


To configure the restore controller, create a manifest file with the definition of [PulpRestore CR](/pulp_operator/restore/#pulprestorespec).
For example:
```
---
apiVersion: repo-manager.pulpproject.org/v1beta2
kind: PulpRestore
metadata:
  name: pulprestore-sample
spec:
  backup_name: pulpbackup-sample
  deployment_name: pulp
```

In the above sample we defined:

* the name of `PulpBackup` instance (`backup_name`)
* the name of `Pulp` instance (`deployment_name`). This should be the same defined in the `PulpBackup` CR.

After finishing to configure the file, apply the configuration and the Operator will start the restore:
```
kubectl apply -f <restore_cr_file>.yaml
```

By default, the restore procedure will reprovision the environment with a single replica of each component. This is to make it easier to review the restore status and the environment health.  
It is also possible to restore with the same number of replicas running when the backup was made. To do so, just set the `keep_replicas` field to true, for example:
```
---
apiVersion: repo-manager.pulpproject.org/v1beta2
kind: PulpRestore
metadata:
  name: pulprestore-sample
spec:
  backup_name: pulpbackup-sample
  deployment_name: pulp
  keep_replicas: true
```


After finishing to restore the environment, the operator will create a `ConfigMap` called *`restore-lock`*. It is used to prevent a new controller reconciliation loop to run and override any data changed/created with the "old" data from backup.  
To allow the restore controller to run again, delete the *restore-lock* `ConfigMap`.