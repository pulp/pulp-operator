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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PulpSpec defines the desired state of Pulp
type PulpSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Define if the operator should stop managing Pulp resources.
	// If set to true, the operator will not execute any task (it will be "disabled").
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Unmanaged bool `json:"unmanaged,omitempty"`

	// Name of the deployment type.
	// +kubebuilder:default:="pulp"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	DeploymentType string `json:"deployment_type,omitempty"`

	// The size of the file storage; for example 100Gi.
	// This field should be used only if file_storage_storage_class is provided
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:storage_type:File"}
	FileStorageSize string `json:"file_storage_size,omitempty"`

	// The file storage access mode.
	// This field should be used only if file_storage_storage_class is provided
	// +kubebuilder:validation:Enum:=ReadWriteMany;ReadWriteOnce
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:storage_type:File","urn:alm:descriptor:com.tectonic.ui:select:ReadWriteMany"}
	FileStorageAccessMode string `json:"file_storage_access_mode,omitempty"`

	// Storage class to use for the file persistentVolumeClaim
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:storage_type:File","urn:alm:descriptor:io.kubernetes:StorageClass"}
	FileStorageClass string `json:"file_storage_storage_class,omitempty"`

	// The secret for Azure compliant object storage configuration.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure secret"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:fieldDependency:storage_type:Azure"}
	ObjectStorageAzureSecret string `json:"object_storage_azure_secret,omitempty"`

	// The secret for S3 compliant object storage configuration.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="S3 secret"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:fieldDependency:storage_type:S3"}
	ObjectStorageS3Secret string `json:"object_storage_s3_secret,omitempty"`

	// PersistenVolumeClaim name that will be used by Pulp pods
	// If defined, the PVC must be provisioned by the user and the operator will only
	// configure the deployment to use it
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:PersistentVolumeClaim","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PVC string `json:"pvc,omitempty"`

	// Secret where the Fernet symmetric encryption key is stored.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Database encryption"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	DBFieldsEncryptionSecret string `json:"db_fields_encryption_secret,omitempty"`

	// Secret where the signing certificates are stored.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SigningSecret string `json:"signing_secret,omitempty"`

	// ConfigMap where the signing scripts are stored.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:ConfigMap","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SigningScriptsConfigmap string `json:"signing_scripts_configmap,omitempty"`

	// Configuration for the storage type utilized in the backup
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=none;File;file;S3;s3;Azure;azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:File","urn:alm:descriptor:com.tectonic.ui:select:S3","urn:alm:descriptor:com.tectonic.ui:select:Azure"}
	StorageType string `json:"storage_type,omitempty"`

	// The ingress type to use to reach the deployed instance
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=none;Ingress;ingress;Route;route;LoadBalancer;loadbalancer;NodePort;nodeport
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:Route","urn:alm:descriptor:com.tectonic.ui:select:Ingress","urn:alm:descriptor:com.tectonic.ui:select:LoadBalancer","urn:alm:descriptor:com.tectonic.ui:select:NodePort"}
	IngressType string `json:"ingress_type,omitempty"`

	// Annotations for the Ingress
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressAnnotations map[string]string `json:"ingress_annotations,omitempty"`

	// Ingress DNS host
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressHost string `json:"ingress_host,omitempty"`

	// Route DNS host
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Route"}
	RouteHost string `json:"route_host,omitempty"`

	// RouteLabels will append custom label(s) into routes (used by router shard routeSelector).
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RouteLabels map[string]string `json:"route_labels,omitempty"`

	// Provide requested port value
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:NodePort"}
	NodePort int32 `json:"nodeport_port,omitempty"`

	// The timeout for HAProxy.
	// +kubebuilder:default:="180s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	HAProxyTimeout string `json:"haproxy_timeout,omitempty"`

	// The client max body size for Nginx Ingress
	// +kubebuilder:default:="10m"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxMaxBodySize string `json:"nginx_client_max_body_size,omitempty"`

	// The proxy body size for Nginx Ingress
	// +kubebuilder:default:="0"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyBodySize string `json:"nginx_proxy_body_size,omitempty"`

	// The proxy read timeout for Nginx Ingress
	// +kubebuilder:default:="120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyReadTimeout string `json:"nginx_proxy_read_timeout,omitempty"`

	// The proxy connect timeout for Nginx Ingress
	// +kubebuilder:default:="120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyConnectTimeout string `json:"nginx_proxy_connect_timeout,omitempty"`

	// The proxy send timeout for Nginx Ingress
	// +kubebuilder:default:="120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxySendTimeout string `json:"nginx_proxy_send_timeout,omitempty"`

	// Secret where the container token certificates are stored.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerTokenSecret string `json:"container_token_secret,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_public_key.pem"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerAuthPublicKey string `json:"container_auth_public_key_name,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_private_key.pem"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerAuthPrivateKey string `json:"container_auth_private_key_name,omitempty"`

	// The image name (repo name) for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp-minimal"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Image string `json:"image,omitempty"`

	// The image version for the pulp image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImageVersion string `json:"image_version,omitempty"`

	// Image pull policy for container image
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=IfNotPresent;Always;Never
	// +kubebuilder:default:="IfNotPresent"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy string `json:"image_pull_policy,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Api Api `json:"api,omitempty"`

	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Database Database `json:"database,omitempty"`

	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Content Content `json:"content,omitempty"`

	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Worker Worker `json:"worker,omitempty"`

	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Web Web `json:"web,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Cache Cache `json:"cache,omitempty"`

	// The pulp settings.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PulpSettings runtime.RawExtension `json:"pulp_settings,omitempty"`

	// The image name (repo name) for the pulp webserver image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp-web"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	ImageWeb string `json:"image_web,omitempty"`

	// The image version for the pulp webserver image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	ImageWebVersion string `json:"image_web_version,omitempty"`

	// Secret where the administrator password can be found
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	AdminPasswordSecret string `json:"admin_password_secret,omitempty"`

	// Image pull secrets for container images
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecrets []string `json:"image_pull_secrets,omitempty"`

	// Secret where Single Sign-on configuration can be found
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SSOSecret string `json:"sso_secret,omitempty"`

	// Define if the operator should or should not mount the custom CA certificates added to the cluster via cluster-wide proxy config
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	TrustedCa bool `json:"mount_trusted_ca,omitempty"`
}

type Api struct {

	// Size is the size of number of pulp-api replicas.
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Optional
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas"`

	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology rule(s) for the pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`

	// The timeout for the gunicorn process.
	// +kubebuilder:default:=90
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// The number of gunicorn workers to use for the api.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornWorkers int `json:"gunicorn_workers,omitempty"`

	// Resource requirements for the pulp api container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	PDB *policy.PodDisruptionBudgetSpec `json:"pdb,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`
}

type Content struct {
	// Size is the size of number of pulp-content replicas
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Optional
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas"`

	// Resource requirements for the pulp-content container
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`

	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology rule(s) for the pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`

	// The timeout for the gunicorn process.
	// +kubebuilder:default:=90
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// The number of gunicorn workers to use for the api.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornWorkers int `json:"gunicorn_workers,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	PDB *policy.PodDisruptionBudgetSpec `json:"pdb,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`
}

type Worker struct {
	// Size is the size of number of pulp-worker replicas
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Optional
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas"`

	// Resource requirements for the pulp-api container
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`

	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology rule(s) for the pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	PDB *policy.PodDisruptionBudgetSpec `json:"pdb,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`
}

type Web struct {
	// Size is the size of number of pulp-web replicas
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Optional
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas"`

	// Resource requirements for the pulp-web container
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// NodeSelector for the Web pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	PDB *policy.PodDisruptionBudgetSpec `json:"pdb,omitempty"`
}

type Database struct {
	// Size is the size of number of db replicas
	// The default postgres image does not provide clustering
	//Replicas int32 `json:"replicas,omitempty"`

	// Secret name with the configuration to use an external database
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ExternalDBSecret string `json:"external_db_secret,omitempty"`

	// PostgreSQL version [default: "13"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresVersion string `json:"version,omitempty"`

	// PostgreSQL port [default: 5432]
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	PostgresPort int `json:"postgres_port,omitempty"`

	// Configure PostgreSQL connection sslmode option [default: "prefer"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresSSLMode string `json:"postgres_ssl_mode,omitempty"`

	// PostgreSQL container image [default: "postgres:13"]
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresImage string `json:"postgres_image,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresExtraArgs []string `json:"postgres_extra_args,omitempty"`

	// Registry path to the PostgreSQL container to use [default: "/var/lib/postgresql/data/pgdata"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresDataPath string `json:"postgres_data_path,omitempty"`

	// Arguments to pass to PostgreSQL initdb command when creating a new cluster. [default: "--auth-host=scram-sha-256"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresInitdbArgs string `json:"postgres_initdb_args,omitempty"`

	// PostgreSQL host authentication method [default: "scram-sha-256"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresHostAuthMethod string `json:"postgres_host_auth_method,omitempty"`

	// Resource requirements for the database container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"postgres_resource_requirements,omitempty"`

	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector for the database pod.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// Node tolerations for the database pod.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Temporarily modifying it as a string to avoid an issue with backup and json.Unmarshal
	// when set as resource.Quantity and no value passed on pulp CR, during backup steps
	// json.Unmarshal is settings it with "0"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresStorageRequirements string `json:"postgres_storage_requirements,omitempty"`

	// Name of the StorageClass required by the claim.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresStorageClass *string `json:"postgres_storage_class,omitempty"`

	// PersistenVolumeClaim name that will be used by database pods
	// If defined, the PVC must be provisioned by the user and the operator will only
	// configure the deployment to use it
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:PersistentVolumeClaim","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PVC string `json:"pvc,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`
}

type Cache struct {

	// Name of the secret with the parameters to connect to an external Redis cluster
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ExternalCacheSecret string `json:"external_cache_secret,omitempty"`

	// Defines if cache should be enabled.
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled,omitempty"`

	// The image name for the redis image. [default: "redis:latest"]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisImage string `json:"redis_image,omitempty"`

	// Storage class to use for the Redis PVC
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass","urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisStorageClass string `json:"redis_storage_class,omitempty"`

	// The port that will be exposed by Redis Service. [default: 6379]
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisPort int `json:"redis_port,omitempty"`

	// Resource requirements for the Redis container
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisResourceRequirements corev1.ResourceRequirements `json:"redis_resource_requirements,omitempty"`

	// PersistenVolumeClaim name that will be used by Redis pods
	// If defined, the PVC must be provisioned by the user and the operator will only
	// configure the deployment to use it
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:PersistentVolumeClaim","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PVC string `json:"pvc,omitempty"`

	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Probe","urn:alm:descriptor:com.tectonic.ui:advanced"}
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`
}

// PulpStatus defines the observed state of Pulp
type PulpStatus struct {
	//+operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions"`
	// Name of the deployment type.
	DeploymentType string `json:"deployment_type,omitempty"`
	// The secret for Azure compliant object storage configuration.
	ObjectStorageAzureSecret string `json:"object_storage_azure_secret,omitempty"`
	// The secret for S3 compliant object storage configuration.
	ObjectStorageS3Secret string `json:"object_storage_s3_secret,omitempty"`
	// Secret where the Fernet symmetric encryption key is stored.
	DBFieldsEncryptionSecret string `json:"db_fields_encryption_secret,omitempty"`
	// The ingress type to use to reach the deployed instance
	IngressType string `json:"ingress_type,omitempty"`
	// Secret where the container token certificates are stored.
	ContainerTokenSecret string `json:"container_token_secret,omitempty"`
	// Secret where the administrator password can be found
	AdminPasswordSecret string `json:"admin_password_secret,omitempty"`
	// Name of the secret with the parameters to connect to an external Redis cluster
	ExternalCacheSecret string `json:"external_cache_secret,omitempty"`
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
