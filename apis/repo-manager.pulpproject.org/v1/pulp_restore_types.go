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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PulpRestoreSpec defines the desired state of PulpRestore
type PulpRestoreSpec struct {

	// Name of Pulp CR to be restored
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="pulp"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DeploymentName string `json:"deployment_name"`

	// Name of PulpBackup CR
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	BackupName string `json:"backup_name"`

	// Name of the PVC to be restored from, set as a status found on the backup object (backupClaim)
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	BackupPVC string `json:"backup_pvc"`

	// Backup directory name, set as a status found on the backup object (backupDirectory)
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	BackupDir string `json:"backup_dir"`

	// KeepBackupReplicasCount allows to define if the restore controller should restore the components with the
	// same number of replicas from backup or restore only a single replica each.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KeepBackupReplicasCount bool `json:"keep_replicas"`
}

// PulpRestoreStatus defines the observed state of PulpRestore
type PulpRestoreStatus struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions"`

	//+operator-sdk:csv:customresourcedefinitions:type=status
	PostgresSecret string `json:"postgres_secret"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// PulpRestore is the Schema for the pulprestores API
type PulpRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PulpRestoreSpec   `json:"spec,omitempty"`
	Status PulpRestoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PulpRestoreList contains a list of PulpRestore
type PulpRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PulpRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PulpRestore{}, &PulpRestoreList{})
}
