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
	DeploymentType string `json:"deployment_type"`

	// The secret for S3 compliant object storage configuration.
	// +kubebuilder:validation:Optional
	ObjectStorageS3Secret string `json:"object_storage_s3_secret"`

	// Secret where the Fernet symmetric encryption key is stored.
	// +kubebuilder:validation:Optional
	DBFieldsEncryptionSecret string `json:"db_fields_encryption_secret"`

	// Secret where the signing certificates are stored.
	// +kubebuilder:validation:Optional
	SigningSecret string `json:"signing_secret"`

	// ConfigMap where the signing scripts are stored.
	// +kubebuilder:validation:Optional
	SigningScriptsConfigmap string `json:"signing_scripts_configmap"`

	// Secret where the container token certificates are stored.
	// +kubebuilder:validation:Optional
	ContainerTokenSecret string `json:"container_token_secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_public_key.pem"
	ContainerAuthPublicKey string `json:"container_auth_public_key_name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_private_key.pem"
	ContainerAuthPrivateKey string `json:"container_auth_private_key_name"`

	// The image name (repo name) for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp"
	Image string `json:"image"`

	// The image version for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	ImageVersion string `json:"image_version"`

	// Image pull policy for container image
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=IfNotPresent;Always;Never
	// +kubebuilder:default:="IfNotPresent"
	ImagePullPolicy string `json:"image_pull_policy"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	IsK8s bool `json:"is_k8s"`

	Api Api `json:"api"`

	//+kubebuilder:validation:Optional
	Database Database `json:"database"`

	//+kubebuilder:validation:Optional
	Content Content `json:"content"`

	//+kubebuilder:validation:Optional
	Worker Worker `json:"worker"`

	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	CacheEnabled bool `json:"cache_enabled,omitempty"`

	// +kubebuilder:default:=6379
	// +kubebuilder:validation:Optional
	RedisPort int `json:"redis_port,omitempty"`

	// +kubebuilder:default:="13"
	// +kubebuilder:validation:Optional
	PostgresVersion string `json:"postgres_version,omitempty"`

	// +kubebuilder:default:=5432
	// +kubebuilder:validation:Optional
	PostgresPort int `json:"postgres_port,omitempty"`
}

type Affinity struct {
	*corev1.NodeAffinity `json:"nodeAffinity,omitempty" protobuf:"bytes,1,opt,name=nodeAffinity"`
}

type Api struct {
	// Size is the size of number of pulp-api replicas.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`

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

	// The pulp settings.
	// +kubebuilder:validation:Optional
	PulpSettings `json:"pulp_settings,omitempty"`

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
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements"`
}

type PulpSettings struct {
	// +kubebuilder:validation:Optional
	Debug string `json:"debug,omitempty"`

	// +kubebuilder:validation:Optional
	GalaxyFeatureFlags `json:"GALAXY_FEATURE_FLAGS,omitempty"`
}

type GalaxyFeatureFlags struct {
	// +kubebuilder:validation:Optional
	ExecutionEnvironments string `json:"execution_environments,omitempty"`
}

type Content struct {
	// Size is the size of number of pulp-content replicas
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`
}

type Worker struct {
	// Size is the size of number of pulp-worker replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32 `json:"replicas"`
}

type Database struct {
	// Size is the size of number of db replicas
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=1
	Replicas int32  `json:"replicas"`
	Type     string `json:"type"`
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
