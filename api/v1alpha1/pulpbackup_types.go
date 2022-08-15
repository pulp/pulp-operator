/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PulpBackupSpec defines the desired state of PulpBackup
type PulpBackupSpec struct {
	DeploymentType string `json:"deployment_type"`

	// Name of the deployment to be backed up
	// +kubebuilder:validation:Optional
	DeploymentName string `json:"deployment_name"`

	// +kubebuilder:default:="pulp"
	InstanceName string `json:"instance_name"`

	// Name of the PVC to be used for storing the backup
	// +kubebuilder:validation:Optional
	BackupPVC string `json:"backup_pvc"`

	// Namespace PVC is in
	// +kubebuilder:validation:Optional
	BackupPVCNamespace string `json:"backup_pvc_namespace"`

	// Storage requirements for the backup
	// +kubebuilder:validation:Optional
	BackupStorageReq string `json:"backup_storage_requirements"`

	// Storage class to use when creating PVC for backup
	// +kubebuilder:validation:Optional
	BackupSC string `json:"backup_storage_class"`

	// Label selector used to identify postgres pod for executing migration
	// +kubebuilder:validation:Optional
	PostgresLabelSelector string `json:"postgres_label_selector"`

	// Secret where the administrator password can be found
	// +kubebuilder:default:="pulp-admin-password"
	AdminPasswordSecret string `json:"admin_password_secret,omitempty"`

	// Secret where the database configuration can be found
	// +kubebuilder:default:="pulp-postgres-configuration"
	PostgresConfigurationSecret string `json:"postgres_configuration_secret"`
}

// PulpBackupStatus defines the observed state of PulpBackup
type PulpBackupStatus struct {
	Conditions []metav1.Condition `json:"conditions"`

	// Name of the deployment backed up
	DeploymentName string `json:"deploymentName"`

	// The PVC name used for the backup
	BackupClaim string `json:"backupClaim"`

	// The namespace used for the backup claim
	BackupNamespace string `json:"backupNamespace"`

	// The directory data is backed up to on the PVC
	BackupDirectory string `json:"backupDirectory"`

	// The deployment storage type
	DeploymentStorageType string `json:"deploymentStorageType"`

	// Administrator password secret used by the deployed instance
	AdminPasswordSecret string `json:"adminPasswordSecret"`

	// Database configuration secret used by the deployed instance
	DatabaseConfigSecret string `json:"databaseConfigurationSecret"`

	// Objectstorage configuration secret used by the deployed instance
	StorageSecret string `json:"storageSecret"`

	// DB fields encryption configuration secret used by deployed instance
	DBFieldsEncryptionSecret string `json:"dbFieldsEncryptionSecret"`

	// Container token configuration secret used by the deployed instance
	ContainerTokenSecret string `json:"containerTokenSecret"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PulpBackup is the Schema for the pulpbackups API
type PulpBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PulpBackupSpec   `json:"spec,omitempty"`
	Status PulpBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PulpBackupList contains a list of PulpBackup
type PulpBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PulpBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PulpBackup{}, &PulpBackupList{})
}
