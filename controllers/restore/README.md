# Pulp Restore

### Custom Resources

* [PulpRestore](#pulprestore)

### Sub Resources

* [PulpRestoreList](#pulprestorelist)
* [PulpRestoreSpec](#pulprestorespec)
* [PulpRestoreStatus](#pulprestorestatus)

#### PulpRestore

PulpRestore is the Schema for the pulprestores API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [PulpRestoreSpec](#pulprestorespec) | false |
| status |  | [PulpRestoreStatus](#pulprestorestatus) | false |

[Back to Custom Resources](#custom-resources)

#### PulpRestoreList

PulpRestoreList contains a list of PulpRestore

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][PulpRestore](#pulprestore) | true |

[Back to Custom Resources](#custom-resources)

#### PulpRestoreSpec

PulpRestoreSpec defines the desired state of PulpRestore

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| deployment_type | Name of the deployment type. Can be one of {galaxy,pulp}. | string | true |
| deployment_name | Name of the deployment to be restored to | string | true |
| backup_name | Name of the backup custom resource | string | true |
| backup_pvc | Name of the PVC to be restored from, set as a status found on the backup object (backupClaim) | string | true |
| backup_dir | Backup directory name, set as a status found on the backup object (backupDirectory) | string | true |
| keep_replicas | KeepBackupReplicasCount allows to define if the restore controller should restore the components with the same number of replicas from backup or restore only a single replica each. | bool | true |

[Back to Custom Resources](#custom-resources)

#### PulpRestoreStatus

PulpRestoreStatus defines the observed state of PulpRestore

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions |  | []metav1.Condition | true |
| postgres_secret |  | string | true |

[Back to Custom Resources](#custom-resources)
