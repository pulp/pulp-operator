# Pulp Backup

### Custom Resources

* [PulpBackup](#pulpbackup)

### Sub Resources

* [PulpBackupList](#pulpbackuplist)
* [PulpBackupSpec](#pulpbackupspec)
* [PulpBackupStatus](#pulpbackupstatus)

#### PulpBackup

PulpBackup is the Schema for the pulpbackups API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [PulpBackupSpec](#pulpbackupspec) | false |
| status |  | [PulpBackupStatus](#pulpbackupstatus) | false |

[Back to Custom Resources](#custom-resources)

#### PulpBackupList

PulpBackupList contains a list of PulpBackup

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][PulpBackup](#pulpbackup) | true |

[Back to Custom Resources](#custom-resources)

#### PulpBackupSpec

PulpBackupSpec defines the desired state of PulpBackup

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| deployment_type | Name of the deployment type. Can be one of {galaxy,pulp}. | string | true |
| deployment_name | Name of the deployment to be backed up | string | true |
| backup_pvc | Name of the PVC to be used for storing the backup | string | true |
| backup_pvc_namespace | Namespace PVC is in | string | true |
| backup_storage_requirements | Storage requirements for the backup | string | true |
| backup_storage_class | Storage class to use when creating PVC for backup | string | true |
| admin_password_secret | Secret where the administrator password can be found | string | false |
| postgres_configuration_secret | Secret where the database configuration can be found | string | true |
| pulp_secret_key | Secret where the Django SECRET_KEY configuration can be found | string | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |

[Back to Custom Resources](#custom-resources)

#### PulpBackupStatus

PulpBackupStatus defines the observed state of PulpBackup

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions |  | []metav1.Condition | true |
| deploymentName | Name of the deployment backed up | string | true |
| backupClaim | The PVC name used for the backup | string | true |
| backupNamespace | The namespace used for the backup claim | string | true |
| backupDirectory | The directory data is backed up to on the PVC | string | true |
| adminPasswordSecret | Administrator password secret used by the deployed instance | string | true |

[Back to Custom Resources](#custom-resources)
