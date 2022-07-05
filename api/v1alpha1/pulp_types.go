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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PulpSpec defines the desired state of Pulp
type PulpSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name of the deployment type.
	//+kubebuilder:default:="pulp"
	DeploymentType string `json:"deployment_type,omitempty"`

	// +kubebuilder:default:=true
	IsFileStorage bool `json:"is_file_storage,omitempty"`

	// The size of the file storage; for example 100Gi.
	FileStorageSize string `json:"file_storage_size,omitempty"`

	// The file storage access mode.
	// +kubebuilder:validation:Enum:=ReadWriteMany;ReadWriteOnce
	FileStorageAccessMode string `json:"file_storage_access_mode,omitempty"`

	// Storage class to use for the file persistentVolumeClaim
	FileStorageClass string `json:"file_storage_storage_class,omitempty"`

	// The secret for S3 compliant object storage configuration.
	// +kubebuilder:validation:Optional
	ObjectStorageS3Secret string `json:"object_storage_s3_secret,omitempty"`

	// Secret where the Fernet symmetric encryption key is stored.
	// +kubebuilder:validation:Optional
	DBFieldsEncryptionSecret string `json:"db_fields_encryption_secret,omitempty"`

	// Secret where the signing certificates are stored.
	// +kubebuilder:validation:Optional
	SigningSecret string `json:"signing_secret,omitempty"`

	// ConfigMap where the signing scripts are stored.
	// +kubebuilder:validation:Optional
	SigningScriptsConfigmap string `json:"signing_scripts_configmap,omitempty"`

	// Secret where the container token certificates are stored.
	// +kubebuilder:validation:Optional
	ContainerTokenSecret string `json:"container_token_secret,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_public_key.pem"
	ContainerAuthPublicKey string `json:"container_auth_public_key_name,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_private_key.pem"
	ContainerAuthPrivateKey string `json:"container_auth_private_key_name,omitempty"`

	// The image name (repo name) for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp"
	Image string `json:"image,omitempty"`

	// The image version for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	ImageVersion string `json:"image_version,omitempty"`

	// Image pull policy for container image
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=IfNotPresent;Always;Never
	// +kubebuilder:default:="IfNotPresent"
	ImagePullPolicy string `json:"image_pull_policy,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	IsK8s bool `json:"is_k8s,omitempty"`

	Api Api `json:"api,omitempty"`

	//+kubebuilder:validation:Optional
	Database Database `json:"database,omitempty"`

	//+kubebuilder:validation:Optional
	Content Content `json:"content,omitempty"`

	//+kubebuilder:validation:Optional
	Worker Worker `json:"worker,omitempty"`

	//+kubebuilder:validation:Optional
	Web Web `json:"web,omitempty"`

	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	CacheEnabled bool `json:"cache_enabled,omitempty"`

	// +kubebuilder:default:=6379
	// +kubebuilder:validation:Optional
	RedisPort int `json:"redis_port,omitempty"`

	// The pulp settings.
	// +kubebuilder:validation:Optional
	PulpSettings `json:"pulp_settings,omitempty"`
}

type Affinity struct {
	*corev1.NodeAffinity `json:"nodeAffinity,omitempty" protobuf:"bytes,1,opt,name=nodeAffinity"`
}

type Api struct {
	// Size is the size of number of pulp-api replicas.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// Defines various deployment affinities.
	// +kubebuilder:validation:Optional
	Affinity Affinity `json:"affinity,omitempty"`

	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology rule(s) for the pods.
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`

	// The timeout for the gunicorn process.
	// +kubebuilder:default:=90
	// +kubebuilder:validation:Optional
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// The number of gunicorn workers to use for the api.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Optional
	GunicornWorkers int `json:"gunicorn_workers,omitempty"`

	// Resource requirements for the pulp content container.
	// +kubebuilder:validation:Optional
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`
}

type PulpSettings struct {
	// +kubebuilder:validation:Optional
	Debug string `json:"debug,omitempty"`

	// +kubebuilder:validation:Optional
	GalaxyFeatureFlags `json:"GALAXY_FEATURE_FLAGS,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="/pulp/"
	ApiRoot string `json:"api_root,omitempty"`
}

type GalaxyFeatureFlags struct {
	// +kubebuilder:validation:Optional
	ExecutionEnvironments string `json:"execution_environments,omitempty"`
}

type Content struct {
	// Size is the size of number of pulp-content replicas
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
}

type Worker struct {
	// Size is the size of number of pulp-worker replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
}

type Web struct {
	// Size is the size of number of pulp-web replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`
}

type Database struct {
	// Size is the size of number of db replicas
	// The default postgres image does not provide clustering
	//Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default:="13"
	// +kubebuilder:validation:Optional
	PostgresVersion string `json:"version,omitempty"`

	// +kubebuilder:default:=5432
	// +kubebuilder:validation:Optional
	PostgresPort int `json:"postgres_port,omitempty"`

	// +kubebuilder:default:="prefer"
	// +kubebuilder:validation:Optional
	PostgresSSLMode string `json:"postgres_ssl_mode,omitempty"`

	// Registry path to the PostgreSQL container to use
	// +kubebuilder:default:="postgres:13"
	// +kubebuilder:validation:Optional
	PostgresImage string `json:"postgres_image,omitempty"`

	// +kubebuilder:default:={}
	// +kubebuilder:validation:Optional
	PostgresExtraArgs []string `json:"postgres_extra_args,omitempty"`

	// +kubebuilder:default:="/var/lib/postgresql/data/pgdata"
	// +kubebuilder:validation:Optional
	PostgresDataPath string `json:"postgres_data_path"`

	// +kubebuilder:default:="--auth-host=scram-sha-256"
	// +kubebuilder:validation:Optional
	PostgresInitdbArgs string `json:"postgres_initdb_args"`

	// +kubebuilder:default:="scram-sha-256"
	// +kubebuilder:validation:Optional
	PostgresHostAuthMethod string `json:"postgres_host_auth_method"`

	// Resource requirements for the database container.
	// +kubebuilder:validation:Optional
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`

	// Defines various deployment affinities.
	// +kubebuilder:validation:Optional
	Affinity Affinity `json:"affinity,omitempty"`

	// NodeSelector for the database pod.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the database pod.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// +kubebuilder:validation:Optional
	PostgresStorageRequirements corev1.ResourceRequirements `json:"postgres_storage_requirements,omitempty"`

	// Name of the StorageClass required by the claim.
	// +kubebuilder:validation:Optional
	PostgresStorageClass *string `json:"postgres_storage_class,omitempty"`
}

// PulpStatus defines the observed state of Pulp
type PulpStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nodes []string `json:"nodes"`
}

// Pulp is the Schema for the pulps API
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
type Pulp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PulpSpec   `json:"spec,omitempty"`
	Status PulpStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PulpList contains a list of Pulp
type PulpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pulp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pulp{}, &PulpList{})
}
