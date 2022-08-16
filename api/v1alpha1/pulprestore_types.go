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

// PulpRestoreSpec defines the desired state of PulpRestore
type PulpRestoreSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=galaxy;pulp
	// +kubebuilder:default:="pulp"
	DeploymentType string `json:"deployment_type"`

	// backup source
	// +kubebuilder:validation:Enum:=CR;PVC
	// +kubebuilder:validation:Optional
	BackupSource string `json:"backup_source"`

	// Name of the deployment to be restored to
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="pulp"
	DeploymentName string `json:"deployment_name"`

	// Name of the backup custom resource
	BackupName string `json:"backup_name"`

	// Name of the PVC to be restored from, set as a status found on the backup object (backupClaim)
	// +kubebuilder:validation:Optional
	BackupPVC string `json:"backup_pvc"`

	// Namespace the PVC is in
	// +kubebuilder:validation:Optional
	BackupPVCNamespace string `json:"backup_pvc_namespace"`

	// Backup directory name, set as a status found on the backup object (backupDirectory)
	// +kubebuilder:validation:Optional
	BackupDir string `json:"backup_dir"`

	// Configuration for the storage type utilized in the backup
	// +kubebuilder:validation:Optional
	StorageType string `json:"storage_type"`

	// Label selector used to identify postgres pod for executing migration
	// +kubebuilder:validation:Optional
	PostgresLabelSelector string `json:"postgres_label_selector"`
}

// PulpRestoreStatus defines the observed state of PulpRestore
type PulpRestoreStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
