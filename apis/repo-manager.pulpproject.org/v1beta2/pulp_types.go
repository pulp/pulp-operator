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

package v1beta2

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PulpSpec defines the desired state of Pulp
type PulpSpec struct {

	/*
		[TODO] List of fields from ansible not "translated" into golang version yet.

		// golang has only pulp_settings defined as raw_settings, which will probably be
		// compatible with ansible version (pending tests)
		pulp_settings.debug
		pulp_settings.GALAXY_FEATURE_FLAGS.execution_environments

		// these seem to be used by migration and upgrade tasks, which are not implemented in go
		// need to understand if these are needed and how they are used (they were added into
		// the spec struct but are not in use yet)
		postgres_migrant_configuration_secret
		postgres_label_selector
		postgres_keep_pvc_after_upgrade

		no_log (not sure if this is useful in golang version)
		resource_manager => need to confirm if this resource is deprecated
	*/

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Define if the operator should stop managing Pulp resources.
	// If set to true, the operator will not execute any task (it will be "disabled").
	// Default: false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Unmanaged bool `json:"unmanaged,omitempty"`

	// Name of the deployment type.
	// Default: "pulp"
	// +kubebuilder:default:="pulp"
	// +kubebuilder:validation:Enum:=pulp;galaxy
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

	// PersistenVolumeClaim name that will be used by Pulp pods.
	// If defined, the PVC must be provisioned by the user and the operator will only
	// configure the deployment to use it
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:PersistentVolumeClaim","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PVC string `json:"pvc,omitempty"`

	// Secret where the Fernet symmetric encryption key is stored.
	// Default: <operators's name>-"-db-fields-encryption"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Database encryption"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	DBFieldsEncryptionSecret string `json:"db_fields_encryption_secret,omitempty"`

	// Secret where the signing certificates are stored.
	// Default: <operators's name>-"-signing-scripts"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SigningSecret string `json:"signing_secret,omitempty"`

	// ConfigMap where the signing scripts are stored.
	// Default: <operators's name>-"-signing-scripts"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:ConfigMap","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SigningScriptsConfigmap string `json:"signing_scripts_configmap,omitempty"`

	// Configuration for the storage type utilized in the backup
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=none;File;file;S3;s3;Azure;azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:File","urn:alm:descriptor:com.tectonic.ui:select:S3","urn:alm:descriptor:com.tectonic.ui:select:Azure"}
	StorageType string `json:"storage_type,omitempty"`

	// The ingress type to use to reach the deployed instance.
	// Default: none (will not expose the service)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=none;Ingress;ingress;Route;route;LoadBalancer;loadbalancer;NodePort;nodeport
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:Route","urn:alm:descriptor:com.tectonic.ui:select:Ingress","urn:alm:descriptor:com.tectonic.ui:select:LoadBalancer","urn:alm:descriptor:com.tectonic.ui:select:NodePort"}
	IngressType string `json:"ingress_type,omitempty"`

	// Annotations for the Ingress
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressAnnotations map[string]string `json:"ingress_annotations,omitempty"`

	// IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress.
	// Default: "" (will use the default ingress class)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressClassName string `json:"ingress_class_name,omitempty"`

	// Define if the IngressClass provided has Nginx as Ingress Controller.
	// If the Ingress Controller is not nginx the operator will automatically provision `pulp-web` pods to redirect the traffic.
	// If it is a nginx controller the traffic will be forwarded to api and content pods.
	// This variable is a workaround to avoid having to grant a ClusterRole (to do a get into the IngressClass and verify the controller).
	// Default: false
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IsNginxIngress bool `json:"is_nginx_ingress,omitempty"`

	// Ingress DNS host
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressHost string `json:"ingress_host,omitempty"`

	// Ingress TLS secret
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	IngressTLSSecret string `json:"ingress_tls_secret,omitempty"`

	// Route DNS host.
	// Default: <operator's name> + "." + ingress.Spec.Domain
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Route"}
	RouteHost string `json:"route_host,omitempty"`

	// RouteLabels will append custom label(s) into routes (used by router shard routeSelector).
	// Default: {"pulp_cr": "<operator's name>", "owner": "pulp-dev" }
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Route"}
	RouteLabels map[string]string `json:"route_labels,omitempty"`

	// RouteAnnotations will append custom annotation(s) into routes (used by router shard routeSelector).
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Route"}
	RouteAnnotations map[string]string `json:"route_annotations,omitempty"`

	// Name of the secret with the certificates/keys used by route encryption
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Route"}
	RouteTLSSecret string `json:"route_tls_secret,omitempty"`

	// Provide requested port value
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:NodePort"}
	NodePort int32 `json:"nodeport_port,omitempty"`

	// The timeout for HAProxy.
	// Default: "180s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:advanced"}
	HAProxyTimeout string `json:"haproxy_timeout,omitempty"`

	// The client max body size for Nginx Ingress.
	// Default: "10m"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxMaxBodySize string `json:"nginx_client_max_body_size,omitempty"`

	// The proxy body size for Nginx Ingress.
	// Default: "0"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyBodySize string `json:"nginx_proxy_body_size,omitempty"`

	// The proxy read timeout for Nginx Ingress.
	// Default: "120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyReadTimeout string `json:"nginx_proxy_read_timeout,omitempty"`

	// The proxy connect timeout for Nginx Ingress.
	// Default: "120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxyConnectTimeout string `json:"nginx_proxy_connect_timeout,omitempty"`

	// The proxy send timeout for Nginx Ingress.
	// Default: "120s"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	NginxProxySendTimeout string `json:"nginx_proxy_send_timeout,omitempty"`

	// Secret where the container token certificates are stored.
	// Default: <operator's name> + "-container-auth"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerTokenSecret string `json:"container_token_secret,omitempty"`

	// Public Key name from `<operator's name> + "-container-auth-certs"` Secret.
	// Default: "container_auth_public_key.pem"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_public_key.pem"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerAuthPublicKey string `json:"container_auth_public_key_name,omitempty"`

	// Private Key name from `<operator's name> + "-container-auth-certs"` Secret.
	// Default: "container_auth_private_key.pem"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="container_auth_private_key.pem"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ContainerAuthPrivateKey string `json:"container_auth_private_key_name,omitempty"`

	// The image name (repo name) for the pulp image.
	// Default: "quay.io/pulp/pulp-minimal:stable"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp-minimal"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Image string `json:"image,omitempty"`

	// The image version for the pulp image.
	// Default: "stable"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImageVersion string `json:"image_version,omitempty"`

	// Relax the check of image_version and image_web_version not matching.
	// Default: "false"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	InhibitVersionConstraint bool `json:"inhibit_version_constraint,omitempty"`

	// Image pull policy for container image.
	// Default: "IfNotPresent"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=IfNotPresent;Always;Never
	// +kubebuilder:default:="IfNotPresent"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy string `json:"image_pull_policy,omitempty"`

	// Api defines desired state of pulpcore-api resources
	// +kubebuilder:default:={replicas:1}
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Api Api `json:"api"`

	// Database defines desired state of postgres resources
	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Database Database `json:"database,omitempty"`

	// Content defines desired state of pulpcore-content resources
	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Content Content `json:"content,omitempty"`

	// Worker defines desired state of pulpcore-worker resources
	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Worker Worker `json:"worker,omitempty"`

	// Web defines desired state of pulpcore-web (reverse-proxy) resources
	//+kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Web Web `json:"web,omitempty"`

	// Cache defines desired state of redis resources
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Cache Cache `json:"cache,omitempty"`

	// Definition of /etc/pulp/settings.py config file.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PulpSettings runtime.RawExtension `json:"pulp_settings,omitempty"`

	// The image name (repo name) for the pulp webserver image.
	// Default: "quay.io/pulp/pulp-web"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="quay.io/pulp/pulp-web"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	ImageWeb string `json:"image_web,omitempty"`

	// The image version for the pulp webserver image.
	// Default: "stable"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="stable"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	ImageWebVersion string `json:"image_web_version,omitempty"`

	// Secret where the administrator password can be found.
	// Default: <operator's name> + "-admin-password"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	AdminPasswordSecret string `json:"admin_password_secret,omitempty"`

	// Image pull secrets for container images.
	// Default: []
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecrets []string `json:"image_pull_secrets,omitempty"`

	// Secret where Single Sign-on configuration can be found
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret","urn:alm:descriptor:com.tectonic.ui:advanced"}
	SSOSecret string `json:"sso_secret,omitempty"`

	// Define if the operator should or should not mount the custom CA certificates added to the cluster via cluster-wide proxy config.
	// Default: false
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	TrustedCa bool `json:"mount_trusted_ca,omitempty"`

	// Define if the operator should or should not deploy the default Execution Environments.
	// Default: false
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	DeployEEDefaults bool `json:"deploy_ee_defaults,omitempty"`

	// Name of the ConfigMap with the list of Execution Environments that should be synchronized.
	// Default: ee-default-images
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	EEDefaults string `json:"ee_defaults,omitempty"`

	/*
	 DEPRECATED FIELDS FROM ANSIBLE VERSION
	*/

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Image pull secret for container images.
	// Default: ""
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecret string `json:"image_pull_secret,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Affinity is a group of affinity scheduling rules.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Affinity *Affinity `json:"affinity,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// The image name for the redis image.
	// Default: "redis:latest"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisImage string `json:"redis_image,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Storage class to use for the Redis PVC
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass","urn:alm:descriptor:com.tectonic.ui:advanced"}
	RedisStorageClass string `json:"redis_storage_class,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Redis defines desired state of cache resources
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Redis Redis `json:"redis,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Storage size to use for the Redis PVC
	RedisStorageSize string `json:"redis_storage_size,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Arguments to pass to PostgreSQL initdb command when creating a new cluster.
	// Default: "--auth-host=scram-sha-256"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresInitdbArgs string `json:"postgres_initdb_args,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// PostgreSQL host authentication method.
	// Default: "scram-sha-256"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresHostAuthMethod string `json:"postgres_host_auth_method,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// PostgreSQL container image.
	// Default: "postgres:13"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresImage string `json:"postgres_image,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Storage resource requirements for the database container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresStorageRequirements *corev1.ResourceRequirements `json:"postgres_storage_requirements,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Resource requirements for the database container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresResourceRequirements *corev1.ResourceRequirements `json:"postgres_resource_requirements,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Name of the StorageClass required by the claim.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass","urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresStorageClass *string `json:"postgres_storage_class,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Registry path to the PostgreSQL container to use.
	// Default: "/var/lib/postgresql/data/pgdata"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresDataPath string `json:"postgres_data_path,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Arguments to pass to postgres process
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresExtraArgs []string `json:"postgres_extra_args,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// The number of gunicorn workers to use for the api.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornAPIWorkers int `json:"gunicorn_api_workers,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// The number of gunicorn workers to use for the api.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornContentWorkers int `json:"gunicorn_content_workers,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// The timeout for the gunicorn process.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// NodeSelector for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NodeSelector map[string]string `json:"node_selector,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Node tolerations for the Pulp pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Node tolerations for the Database pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresTolerations []corev1.Toleration `json:"postgres_tolerations,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Topology rule(s) for the pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topology_spread_constraints,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Custom configuration secret of database pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresConfigurationSecret string `json:"postgres_configuration_secret,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// NodeSelector for the database pod.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresSelector map[string]string `json:"postgres_selector,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Ingress DNS host
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text","urn:alm:descriptor:com.tectonic.ui:fieldDependency:ingress_type:Ingress"}
	Hostname string `json:"hostname,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// pulp-web service annotations
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ServiceAnnotations string `json:"service_annotations,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// The secure TLS termination mechanism to use in pulp-web pods.
	// Default: "edge"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=edge;Edge;passthrough;Passthrough
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RouteTLSTerminationMechanism string `json:"route_tls_termination_mechanism,omitempty"`

	// Protocol used by pulp-web service when ingress_type==loadbalancer
	// +kubebuilder:validation:Enum:=http;https
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	LoadbalancerProtocol string `json:"loadbalancer_protocol,omitempty"`

	// Port exposed by pulp-web service when ingress_type==loadbalancer
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	LoadbalancerPort int32 `json:"loadbalancer_port,omitempty"`

	// [TODO] Pending implementation, added just to keep compatibility
	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Secret where the old database configuration can be found for data migration
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresMigrantConfigurationSecret string `json:"postgres_migrant_configuration_secret,omitempty"`

	// [TODO] Pending implementation, added just to keep compatibility
	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Label selector used to identify postgres pod for executing migration
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresLabelSelector string `json:"postgres_label_selector,omitempty"`

	// [TODO] Pending implementation, added just to keep compatibility
	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	// Specify whether or not to keep the old PVC after PostgreSQL upgrades
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	PostgresKeepPvcAfterUpgrade bool `json:"postgres_keep_pvc_after_upgrade,omitempty"`

	// Configure no_log for no_log tasks
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	NoLog bool `json:"no_log,omitempty"`

	// Telemetry defines the OpenTelemetry configuration
	// +kubebuilder:validation:Optional
	Telemetry Telemetry `json:"telemetry,omitempty"`

	ResourceManager ResourceManager `json:"resource_manager,omitempty"`
}

// Api defines desired state of pulpcore-api resources
type Api struct {

	// Size is the size of number of pulp-api replicas.
	// Default: 1
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
	// Default: 90
	// +kubebuilder:default:=90
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// The number of gunicorn workers to use for the api.
	// Default: 2
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

	// [TODO] I couldnt find a reference for this var in ansible
	// [DEPRECATED?] Temporarily adding to keep compatibility with ansible version.
	LogLevel string `json:"log_level,omitempty"`

	// InitContainer defines configuration of the init-containers that run in pulpcore pods
	InitContainer InitContainer `json:"init_container,omitempty"`
}

// Content defines desired state of pulpcore-content resources
type Content struct {
	// Size is the size of number of pulp-content replicas.
	// Default: 2
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
	// Default: 90
	// +kubebuilder:default:=90
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	GunicornTimeout int `json:"gunicorn_timeout,omitempty"`

	// The number of gunicorn workers to use for the api.
	// Default: 2
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

	// [TODO] Implement this like in ansible version
	// [DEPRECATED?] Temporarily adding to keep compatibility with ansible version.
	LogLevel string `json:"log_level,omitempty"`

	// InitContainer defines configuration of the init-containers that run in pulpcore pods
	InitContainer InitContainer `json:"init_container,omitempty"`
}

// Worker defines desired state of pulpcore-worker resources
type Worker struct {
	// Size is the size of number of pulp-worker replicas.
	// Default: 2
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

	// InitContainer defines configuration of the init-containers that run in pulpcore pods
	InitContainer InitContainer `json:"init_container,omitempty"`
}

// Web defines desired state of pulpcore-web (reverse-proxy) resources
type Web struct {
	// Size is the size of number of pulp-web replicas.
	// Default: 1
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

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`

	// Annotations for the service
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	ServiceAnnotations map[string]string `json:"service_annotations,omitempty"`

	// The secure TLS termination mechanism to use
	// Default: "edge"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:=edge;Edge;passthrough;Passthrough
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	TLSTerminationMechanism string `json:"tls_termination_mechanism,omitempty"`
}

// Database defines desired state of postgres
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

	// PostgreSQL port.
	// Default: 5432
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	PostgresPort int `json:"postgres_port,omitempty"`

	// Configure PostgreSQL connection sslmode option.
	// Default: "prefer"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresSSLMode string `json:"postgres_ssl_mode,omitempty"`

	// PostgreSQL container image.
	// Default: "postgres:13"
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresImage string `json:"postgres_image,omitempty"`

	// Arguments to pass to postgres process
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresExtraArgs []string `json:"postgres_extra_args,omitempty"`

	// Registry path to the PostgreSQL container to use.
	// Default: "/var/lib/postgresql/data/pgdata"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresDataPath string `json:"postgres_data_path,omitempty"`

	// Arguments to pass to PostgreSQL initdb command when creating a new cluster.
	// Default: "--auth-host=scram-sha-256"
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PostgresInitdbArgs string `json:"postgres_initdb_args,omitempty"`

	// PostgreSQL host authentication method.
	// Default: "scram-sha-256"
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

// Cache defines desired state of redis resources
type Cache struct {

	// Name of the secret with the parameters to connect to an external Redis cluster
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ExternalCacheSecret string `json:"external_cache_secret,omitempty"`

	// Defines if cache should be enabled.
	// Default: true
	// +kubebuilder:default:=true
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled,omitempty"`

	// The image name for the redis image.
	// Default: "redis:latest"
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

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
type Affinity struct {
	NodeAffinity *corev1.NodeAffinity `json:"node_affinity,omitempty"`
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
type Redis struct {
	// [TODO] Implement this like in ansible version
	// [DEPRECATED?] Temporarily adding to keep compatibility with ansible version.
	LogLevel string `json:"log_level,omitempty"`

	// [TODO] THIS SHOULD BE REMOVED SINCE WE WILL NOT SUPPORT REDIS CLUSTER
	// Keeping it just as a matter of compatibility
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas"`

	RedisResourceRequirements *corev1.ResourceRequirements `json:"redis_resource_requirements,omitempty"`
	ResourceRequirements      *corev1.ResourceRequirements `json:"resource_requirements,omitempty"`
	Strategy                  *appsv1.DeploymentStrategy   `json:"strategy,omitempty"`
}

// [DEPRECATED] Temporarily adding to keep compatibility with ansible version
type ResourceManager struct {
	Replicas             int32                        `json:"replicas"`
	Strategy             *appsv1.DeploymentStrategy   `json:"strategy,omitempty"`
	ResourceRequirements *corev1.ResourceRequirements `json:"resource_requirements,omitempty"`
}

// Telemetry defines the configuration for OpenTelemetry used by Pulp
type Telemetry struct {

	// Enable Pulp Telemetry
	// Default: false
	// +kubebuilder:default:=false
	// +kubebuilder:validation:Optional
	// +nullable
	Enabled bool `json:"enabled,omitempty"`

	// Defines the protocol used by the instrumentator to comunicate with the collector
	// Default: http/protobuf
	// +kubebuilder:default:="http/protobuf"
	ExporterOtlpProtocol string `json:"exporter_otlp_protocol,omitempty"`

	// Defines the image to be used as collector
	OpenTelemetryCollectorImage string `json:"otel_collector_image,omitempty"`

	// The image version for opentelemetry-collector image. Default: \"latest\"
	OpenTelemetryCollectorImageVersion string `json:"otel_collector_image_version,omitempty"`

	// Resource requirements for the sidecar container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`
}

// InitContainer defines configuration of the init-containers that run in pulpcore pods
type InitContainer struct {

	// The image name for the init-container.
	// By default, if not provided, it will use the same image from .Spec.Image
	// WARN: defining a different image than the one used by API pods can cause unexpected behaviors!
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Image string `json:"image,omitempty"`

	// Resource requirements for pulpcore init-container.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ResourceRequirements corev1.ResourceRequirements `json:"resource_requirements,omitempty"`
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
	// Name of pulp image deployed.
	Image string `json:"image,omitempty"`
	// The ingress type to use to reach the deployed instance
	IngressType string `json:"ingress_type,omitempty"`
	// IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress.
	IngressClassName string `json:"ingress_class_name,omitempty"`
	// Secret where the container token certificates are stored.
	ContainerTokenSecret string `json:"container_token_secret,omitempty"`
	// Secret where the administrator password can be found
	AdminPasswordSecret string `json:"admin_password_secret,omitempty"`
	// Name of the secret with the parameters to connect to an external Redis cluster
	ExternalCacheSecret string `json:"external_cache_secret,omitempty"`
	// Pulp metrics collection enabled
	TelemetryEnabled bool `json:"telemetry_enabled,omitempty"`

	// [DEPRECATED] Temporarily adding to keep compatibility with ansible version.
	StoragePersistentVolumeClaim       string `json:"storagePersistentVolumeClaim,omitempty"`
	WebURL                             string `json:"webURL,omitempty"`
	DatabaseConfigurationSecret        string `json:"databaseConfigurationSecret,omitempty"`
	StorageType                        string `json:"storageType,omitempty"`
	StorageSecret                      string `json:"storageSecret,omitempty"`
	DeployedVersion                    string `json:"deployedVersion,omitempty"`
	DeployedImage                      string `json:"deployedImage,omitempty"`
	MigrantDatabaseConfigurationSecret string `json:"migrantDatabaseConfigurationSecret,omitempty"`
	DbFieldsEncryptionSecret           string `json:"dbFieldsEncryptionSecret,omitempty"`
	UpgradedPostgresVersion            string `json:"upgradedPostgresVersion,omitempty"`
	MigrationDone                      bool   `json:"migration_done,omitempty"`
}

// Pulp is the Schema for the pulps API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:storageversion

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
