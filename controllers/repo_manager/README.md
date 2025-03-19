
### Custom Resources

* [Pulp](#pulp)

### Sub Resources

* [Api](#api)
* [Cache](#cache)
* [Content](#content)
* [Database](#database)
* [LDAP](#ldap)
* [PulpContainer](#pulpcontainer)
* [PulpJob](#pulpjob)
* [PulpList](#pulplist)
* [PulpSpec](#pulpspec)
* [PulpStatus](#pulpstatus)
* [Telemetry](#telemetry)
* [Web](#web)
* [Worker](#worker)

#### Api

Api defines desired state of pulpcore-api resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-api replicas. Default: 1 | int32 | true |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| gunicorn_timeout | The timeout for the gunicorn process. Default: 90 | int | false |
| gunicorn_workers | The number of gunicorn workers to use for the api. Default: 2 | int | false |
| resource_requirements | Resource requirements for the pulp api container. | corev1.ResourceRequirements | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |
| init_container | InitContainer defines configuration of the init-containers that run in pulpcore pods | [PulpContainer](#pulpcontainer) | false |
| env_vars | Environment variables to add to pulpcore-api container | []corev1.EnvVar | false |
| deployment_annotations | Annotations for the api deployment | map[string]string | false |

[Back to Custom Resources](#custom-resources)

#### Cache

Cache defines desired state of redis resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| external_cache_secret | Name of the secret with the parameters to connect to an external Redis cluster | string | false |
| enabled | Defines if cache should be enabled. Default: true | bool | false |
| redis_image | The image name for the redis image. Default: \"redis:latest\" | string | false |
| redis_storage_class | Storage class to use for the Redis PVC | string | false |
| redis_port | The port that will be exposed by Redis Service. [default: 6379] | int | false |
| redis_resource_requirements | Resource requirements for the Redis container | corev1.ResourceRequirements | false |
| pvc | PersistenVolumeClaim name that will be used by Redis pods If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |
| deployment_annotations | Annotations for the cache deployment | map[string]string | false |

[Back to Custom Resources](#custom-resources)

#### Content

Content defines desired state of pulpcore-content resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-content replicas. Default: 1 | int32 | true |
| resource_requirements | Resource requirements for the pulp-content container | corev1.ResourceRequirements | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| gunicorn_timeout | The timeout for the gunicorn process. Default: 90 | int | false |
| gunicorn_workers | The number of gunicorn workers to use for the api. Default: 2 | int | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |
| init_container | InitContainer defines configuration of the init-containers that run in pulpcore pods | [PulpContainer](#pulpcontainer) | false |
| env_vars | Environment variables to add to pulpcore-content container | []corev1.EnvVar | false |
| deployment_annotations | Annotations for the content deployment | map[string]string | false |

[Back to Custom Resources](#custom-resources)

#### Database

Database defines desired state of postgres

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| external_db_secret | Secret name with the configuration to use an external database | string | false |
| version | PostgreSQL version [default: \"13\"] | string | false |
| postgres_port | PostgreSQL port. Default: 5432 | int | false |
| postgres_ssl_mode | Configure PostgreSQL connection sslmode option. Default: \"prefer\" | string | false |
| postgres_image | PostgreSQL container image. Default: \"postgres:13\" | string | false |
| postgres_extra_args | Arguments to pass to postgres process | []string | false |
| postgres_data_path | Registry path to the PostgreSQL container to use. Default: \"/var/lib/postgresql/data/pgdata\" | string | false |
| postgres_initdb_args | Arguments to pass to PostgreSQL initdb command when creating a new cluster. Default: \"--auth-host=scram-sha-256\" | string | false |
| postgres_host_auth_method | PostgreSQL host authentication method. Default: \"scram-sha-256\" | string | false |
| postgres_resource_requirements | Resource requirements for the database container. | corev1.ResourceRequirements | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| node_selector | NodeSelector for the database pod. | map[string]string | false |
| tolerations | Node tolerations for the database pod. | []corev1.Toleration | false |
| postgres_storage_requirements | Temporarily modifying it as a string to avoid an issue with backup and json.Unmarshal when set as resource.Quantity and no value passed on pulp CR, during backup steps json.Unmarshal is settings it with \"0\" | string | false |
| postgres_storage_class | Name of the StorageClass required by the claim. | *string | false |
| pvc | PersistenVolumeClaim name that will be used by database pods If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |

[Back to Custom Resources](#custom-resources)

#### LDAP

LDAP defines the ldap resources used by pulpcore containers to integrate Pulp with LDAP authentication

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| config | The name of the Secret with ldap config. | string | false |
| ca | The name of the Secret with the CA chain to connect to ldap server. | string | false |

[Back to Custom Resources](#custom-resources)

#### Pulp

Pulp is the Schema for the pulps API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [PulpSpec](#pulpspec) | false |
| status |  | [PulpStatus](#pulpstatus) | false |

[Back to Custom Resources](#custom-resources)

#### PulpContainer

PulpContainer defines configuration of the \"auxiliary\" containers that run in pulpcore pods

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| image | The image name for the container. By default, if not provided, it will use the same image from .Spec.Image. WARN: defining a different image than the one used by API pods can cause unexpected behaviors! | string | false |
| resource_requirements | Resource requirements for pulpcore aux container. | corev1.ResourceRequirements | false |
| env_vars | Environment variables to add to the container | []corev1.EnvVar | false |

[Back to Custom Resources](#custom-resources)

#### PulpJob

PulpJob defines the jobs used by pulpcore containers to run single-shot administrative tasks

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| container |  | [PulpContainer](#pulpcontainer) | false |

[Back to Custom Resources](#custom-resources)

#### PulpList

PulpList contains a list of Pulp

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][Pulp](#pulp) | true |

[Back to Custom Resources](#custom-resources)

#### PulpSpec

PulpSpec defines the desired state of Pulp

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| unmanaged | Define if the operator should stop managing Pulp resources. If set to true, the operator will not execute any task (it will be \"disabled\"). Default: false | bool | false |
| enable_debugging | By default Pulp logs at INFO level, but enabling DEBUG logging can be a helpful thing to get more insight when things don’t go as expected. Default: false | bool | false |
| file_storage_size | The size of the file storage; for example 100Gi. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_access_mode | The file storage access mode. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_storage_class | Storage class to use for the file persistentVolumeClaim | string | false |
| object_storage_azure_secret | The secret for Azure compliant object storage configuration. | string | false |
| object_storage_s3_secret | The secret for S3 compliant object storage configuration. | string | false |
| pvc | PersistenVolumeClaim name that will be used by Pulp pods. If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| db_fields_encryption_secret | Secret where the Fernet symmetric encryption key is stored. Default: <operators's name>-\"-db-fields-encryption\" | string | false |
| signing_secret | Name of the Secret where the gpg key is stored. | string | false |
| signing_scripts | Name of the Secret where the signing scripts are stored. | string | false |
| ingress_type | The ingress type to use to reach the deployed instance. Default: none (will not expose the service) | string | false |
| ingress_annotations | Annotations for the Ingress | map[string]string | false |
| ingress_class_name | IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress. Default: \"\" (will use the default ingress class) | string | false |
| is_nginx_ingress | Define if the IngressClass provided has Nginx as Ingress Controller. If the Ingress Controller is not nginx the operator will automatically provision `pulp-web` pods to redirect the traffic. If it is a nginx controller the traffic will be forwarded to api and content pods. This variable is a workaround to avoid having to grant a ClusterRole (to do a get into the IngressClass and verify the controller). Default: false | bool | false |
| ingress_host | Ingress DNS host | string | false |
| ingress_tls_secret | Ingress TLS secret | string | false |
| route_host | Route DNS host. Default: <operator's name> + \".\" + ingress.Spec.Domain | string | false |
| route_labels | RouteLabels will append custom label(s) into routes (used by router shard routeSelector). Default: {\"pulp_cr\": \"<operator's name>\", \"owner\": \"pulp-dev\" } | map[string]string | false |
| route_annotations | RouteAnnotations will append custom annotation(s) into routes (used by router shard routeSelector). | map[string]string | false |
| route_tls_secret | Name of the secret with the certificates/keys used by route encryption | string | false |
| nodeport_port | Provide requested port value | int32 | false |
| haproxy_timeout | The timeout for HAProxy. Default: \"180s\" | string | false |
| nginx_client_max_body_size | The client max body size for Nginx Ingress. Default: \"10m\" | string | false |
| nginx_proxy_body_size | The proxy body size for Nginx Ingress. Default: \"0\" | string | false |
| nginx_proxy_read_timeout | The proxy read timeout for Nginx Ingress. Default: \"120s\" | string | false |
| nginx_proxy_connect_timeout | The proxy connect timeout for Nginx Ingress. Default: \"120s\" | string | false |
| nginx_proxy_send_timeout | The proxy send timeout for Nginx Ingress. Default: \"120s\" | string | false |
| container_token_secret | Secret where the container token certificates are stored. Default: <operator's name> + \"-container-auth\" | string | false |
| container_auth_public_key_name | Public Key name from `<operator's name> + \"-container-auth-certs\"` Secret. Default: \"container_auth_public_key.pem\" | string | false |
| container_auth_private_key_name | Private Key name from `<operator's name> + \"-container-auth-certs\"` Secret. Default: \"container_auth_private_key.pem\" | string | false |
| image | The image name (repo name) for the pulp image. Default: \"quay.io/pulp/pulp-minimal:stable\" | string | false |
| image_version | The image version for the pulp image. Default: \"stable\" | string | false |
| inhibit_version_constraint | Relax the check of image_version and image_web_version not matching. Default: \"false\" | bool | false |
| image_pull_policy | Image pull policy for container image. | string | false |
| api | Api defines desired state of pulpcore-api resources | [Api](#api) | true |
| database | Database defines desired state of postgres resources | [Database](#database) | false |
| content | Content defines desired state of pulpcore-content resources | [Content](#content) | false |
| worker | Worker defines desired state of pulpcore-worker resources | [Worker](#worker) | false |
| web | Web defines desired state of pulpcore-web (reverse-proxy) resources | [Web](#web) | false |
| cache | Cache defines desired state of redis resources | [Cache](#cache) | false |
| custom_pulp_settings | Name of the ConfigMap to define Pulp configurations not available through this CR. | string | false |
| image_web | The image name (repo name) for the pulp webserver image. Default: \"quay.io/pulp/pulp-web\" | string | false |
| image_web_version | The image version for the pulp webserver image. Default: \"stable\" | string | false |
| admin_password_secret | Secret where the administrator password can be found. Default: <operator's name> + \"-admin-password\" | string | false |
| image_pull_secrets | Image pull secrets for container images. Default: [] | []string | false |
| sa_annotations | ServiceAccount.metadata.annotations that will be used in Pulp pods. | map[string]string | false |
| sa_labels | ServiceAccount.metadata.labels that will be used in Pulp pods. | map[string]string | false |
| sso_secret | Secret where Single Sign-on configuration can be found | string | false |
| mount_trusted_ca | Define if the operator should or should not mount the custom CA certificates added to the cluster via cluster-wide proxy config. Default: false | bool | false |
| admin_password_job | Job to reset pulp admin password | [PulpJob](#pulpjob) | false |
| migration_job | Job to run django migrations | [PulpJob](#pulpjob) | false |
| signing_job | Job to store signing metadata scripts | [PulpJob](#pulpjob) | false |
| disable_migrations | Disable database migrations. Useful for situations in which we don't want to automatically run the database migrations, for example, during restore. | bool | false |
| pulp_secret_key | Name of the Secret to provide Django cryptographic signing. Default: \"pulp-secret-key\" | string | false |
| allowed_content_checksums | List of allowed checksum algorithms used to verify repository's integrity. Valid options: [\"md5\",\"sha1\",\"sha224\",\"sha256\",\"sha384\",\"sha512\"]. | []string | false |
| loadbalancer_protocol | Protocol used by pulp-web service when ingress_type==loadbalancer | string | false |
| loadbalancer_port | Port exposed by pulp-web service when ingress_type==loadbalancer | int32 | false |
| telemetry | Telemetry defines the OpenTelemetry configuration | [Telemetry](#telemetry) | false |
| ldap | LDAP defines the ldap resources used by pulpcore containers to integrate Pulp with LDAP authentication | [LDAP](#ldap) | false |
| ipv6_disabled | Disable ipv6 for pulpcore and pulp-web pods | *bool | false |

[Back to Custom Resources](#custom-resources)

#### PulpStatus

PulpStatus defines the observed state of Pulp

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions |  | []metav1.Condition | true |
| object_storage_azure_secret | The secret for Azure compliant object storage configuration. | string | false |
| object_storage_s3_secret | The secret for S3 compliant object storage configuration. | string | false |
| db_fields_encryption_secret | Secret where the Fernet symmetric encryption key is stored. | string | false |
| image | Name of pulp image deployed. | string | false |
| ingress_type | The ingress type to use to reach the deployed instance | string | false |
| ingress_class_name | IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress. | string | false |
| container_token_secret | Secret where the container token certificates are stored. | string | false |
| admin_password_secret | Secret where the administrator password can be found | string | false |
| external_cache_secret | Name of the secret with the parameters to connect to an external Redis cluster | string | false |
| telemetry_enabled | Pulp metrics collection enabled | bool | false |
| pulp_secret_key | Name of the Secret to provide Django cryptographic signing. | string | false |
| allowed_content_checksums | List of allowed checksum algorithms used to verify repository's integrity. | string | false |
| last_deployment_update | Controller status to keep tracking of deployment updates | string | false |
| managed_cache_enabled | Cache deployed by pulp-operator enabled | bool | false |
| storage_type | Type of storage in use by pulpcore pods | string | false |

[Back to Custom Resources](#custom-resources)

#### Telemetry

Telemetry defines the configuration for OpenTelemetry used by Pulp

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| enabled | Enable Pulp Telemetry Default: false | bool | false |
| exporter_otlp_protocol | Defines the protocol used by the instrumentator to comunicate with the collector Default: http/protobuf | string | false |
| otel_collector_image | Defines the image to be used as collector | string | false |
| otel_collector_image_version | The image version for opentelemetry-collector image. Default: \"latest\" | string | false |
| resource_requirements | Resource requirements for the sidecar container. | corev1.ResourceRequirements | false |

[Back to Custom Resources](#custom-resources)

#### Web

Web defines desired state of pulpcore-web (reverse-proxy) resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-web replicas. Default: 1 | int32 | true |
| resource_requirements | Resource requirements for the pulp-web container | corev1.ResourceRequirements | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| node_selector | NodeSelector for the Web pods. | map[string]string | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |
| service_annotations | Annotations for the service | map[string]string | false |
| tls_termination_mechanism | The secure TLS termination mechanism to use Default: \"edge\" | string | false |
| env_vars | Environment variables to add to pulpcore-web container | []corev1.EnvVar | false |
| deployment_annotations | Annotations for the web deployment | map[string]string | false |

[Back to Custom Resources](#custom-resources)

#### Worker

Worker defines desired state of pulpcore-worker resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-worker replicas. Default: 1 | int32 | true |
| resource_requirements | Resource requirements for the pulp-api container | corev1.ResourceRequirements | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |
| init_container | InitContainer defines configuration of the init-containers that run in pulpcore pods | [PulpContainer](#pulpcontainer) | false |
| env_vars | Environment variables to add to pulpcore-worker container | []corev1.EnvVar | false |
| deployment_annotations | Annotations for the worker deployment | map[string]string | false |

[Back to Custom Resources](#custom-resources)
