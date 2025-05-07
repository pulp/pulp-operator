# Overview of Backup/Restore Operations


In addition to provisioning Pulp components, the Operator can also be used to backup and restore them.

!!! note
    Before starting a backup, make sure that the namespace has enough storage quota available.

## Backup
The backup procedure creates a *manager* `Pod` which will be used to execute all the backup tasks:

* run a `pg_dump` (database dump) on Pulp's database
* do a copy of the Pulp CR instance defined in `deployment_name`
* do a copy of the `Secrets`
* do a copy of `/var/lib/pulp` directory
* delete the *manager* `Pod` to not consume resources

These data will be stored in a new PVC defined in PulpBackup CR (`backup_pvc` or `backup_storage_class`).


!!! note
    The current version of the Operator does **not** execute backups of **external** PostgreSQL instances yet.


!!! notes
    Considering that files stored in Object Storage (like `AWS S3` and/or `Azure Blob`) are not kept in `/var/lib/pulp` directory, they will **not** be copied. If you still need to do a backup of the artifacts stored in Object Storage, please, contact your cloud provider to check the procedure to do so.


## Restore
The restore procedure also creates a *manager* `Pod` to execute all the tasks:

* restore the `Secrets`
* restore Pulp CR instance
* restore Pulp database
* restore `/var/lib/pulp` directory
* delete the *manager* `Pod` to not consume resources

All data restored comes from the PVC defined in PulpRestore CR (`backup_pvc`).
