
### Custom Resources

* [Pulp](#pulp)

### Sub Resources

* [Affinity](#affinity)
* [Api](#api)
* [Cache](#cache)
* [Content](#content)
* [Database](#database)
* [PulpList](#pulplist)
* [PulpSpec](#pulpspec)
* [PulpStatus](#pulpstatus)
* [Redis](#redis)
* [Web](#web)
* [Worker](#worker)

#### Affinity

[DEPRECATED] Temporarily adding to keep compatibility with ansible version

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| node_affinity |  | *corev1.NodeAffinity | false |

[Back to Custom Resources](#custom-resources)

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
| log_level | [TODO] I couldnt find a reference for this var in ansible [DEPRECATED?] Temporarily adding to keep compatibility with ansible version. | string | false |

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

[Back to Custom Resources](#custom-resources)

#### Content

Content defines desired state of pulpcore-content resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-content replicas. Default: 2 | int32 | true |
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
| log_level | [TODO] Implement this like in ansible version [DEPRECATED?] Temporarily adding to keep compatibility with ansible version. | string | false |

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

#### Pulp

Pulp is the Schema for the pulps API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [PulpSpec](#pulpspec) | false |
| status |  | [PulpStatus](#pulpstatus) | false |

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
| deployment_type | Name of the deployment type. Default: \"pulp\" | string | false |
| file_storage_size | The size of the file storage; for example 100Gi. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_access_mode | The file storage access mode. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_storage_class | Storage class to use for the file persistentVolumeClaim | string | false |
| object_storage_azure_secret | The secret for Azure compliant object storage configuration. | string | false |
| object_storage_s3_secret | The secret for S3 compliant object storage configuration. | string | false |
| pvc | PersistenVolumeClaim name that will be used by Pulp pods. If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| db_fields_encryption_secret | Secret where the Fernet symmetric encryption key is stored. Default: <operators's name>-\"-db-fields-encryption\" | string | false |
| signing_secret | Secret where the signing certificates are stored. Default: <operators's name>-\"-signing-scripts\" | string | false |
| signing_scripts_configmap | ConfigMap where the signing scripts are stored. Default: <operators's name>-\"-signing-scripts\" | string | false |
| storage_type | Configuration for the storage type utilized in the backup | string | false |
| ingress_type | The ingress type to use to reach the deployed instance. Default: none (will not expose the service) | string | false |
| ingress_annotations | Annotations for the Ingress | map[string]string | false |
| ingress_class_name | IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress. Default: \"\" (will use the default ingress class) | string | false |
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
| image_pull_policy | Image pull policy for container image. Default: \"IfNotPresent\" | string | false |
| api | Api defines desired state of pulpcore-api resources | [Api](#api) | true |
| database | Database defines desired state of postgres resources | [Database](#database) | false |
| content | Content defines desired state of pulpcore-content resources | [Content](#content) | false |
| worker | Worker defines desired state of pulpcore-worker resources | [Worker](#worker) | false |
| web | Web defines desired state of pulpcore-web (reverse-proxy) resources | [Web](#web) | false |
| cache | Cache defines desired state of redis resources | [Cache](#cache) | false |
| pulp_settings | Definition of /etc/pulp/settings.py config file. | runtime.RawExtension | false |
| image_web | The image name (repo name) for the pulp webserver image. Default: \"quay.io/pulp/pulp-web\" | string | false |
| image_web_version | The image version for the pulp webserver image. Default: \"stable\" | string | false |
| admin_password_secret | Secret where the administrator password can be found. Default: <operator's name> + \"-admin-password\" | string | false |
| image_pull_secrets | Image pull secrets for container images. Default: [] | []string | false |
| sso_secret | Secret where Single Sign-on configuration can be found | string | false |
| mount_trusted_ca | Define if the operator should or should not mount the custom CA certificates added to the cluster via cluster-wide proxy config. Default: false | bool | false |
| deploy_ee_defaults | Define if the operator should or should not deploy the default Execution Environments. Default: false | bool | false |
| ee_defaults | Name of the ConfigMap with the list of Execution Environments that should be synchronized. Default: ee-default-images | string | false |
| image_pull_secret | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Image pull secret for container images. Default: \"\" | string | false |
| affinity | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Affinity is a group of affinity scheduling rules. | *[Affinity](#affinity) | false |
| redis_image | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. The image name for the redis image. Default: \"redis:latest\" | string | false |
| redis_storage_class | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Storage class to use for the Redis PVC | string | false |
| redis | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Redis defines desired state of cache resources | [Redis](#redis) | false |
| postgres_initdb_args | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Arguments to pass to PostgreSQL initdb command when creating a new cluster. Default: \"--auth-host=scram-sha-256\" | string | false |
| postgres_host_auth_method | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. PostgreSQL host authentication method. Default: \"scram-sha-256\" | string | false |
| postgres_image | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. PostgreSQL container image. Default: \"postgres:13\" | string | false |
| postgres_storage_requirements | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Storage resource requirements for the database container. | *corev1.ResourceRequirements | false |
| postgres_resource_requirements | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Resource requirements for the database container. | *corev1.ResourceRequirements | false |
| postgres_storage_class | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Name of the StorageClass required by the claim. | *string | false |
| postgres_data_path | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Registry path to the PostgreSQL container to use. Default: \"/var/lib/postgresql/data/pgdata\" | string | false |
| postgres_extra_args | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Arguments to pass to postgres process | []string | false |
| gunicorn_api_workers | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. The number of gunicorn workers to use for the api. | int | false |
| gunicorn_content_workers | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. The number of gunicorn workers to use for the api. | int | false |
| gunicorn_timeout | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. The timeout for the gunicorn process. | int | false |
| node_selector | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| postgres_tolerations | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Node tolerations for the Database pods. | []corev1.Toleration | false |
| topology_spread_constraints | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| postgres_configuration_secret | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Custom configuration secret of database pods | string | false |
| postgres_selector | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. NodeSelector for the database pod. | map[string]string | false |
| hostname | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. Ingress DNS host | string | false |
| service_annotations | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. pulp-web service annotations | string | false |
| route_tls_termination_mechanism | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. The secure TLS termination mechanism to use in pulp-web pods. Default: \"edge\" | string | false |
| loadbalancer_protocol | Protocol used by pulp-web service when ingress_type==loadbalancer | string | false |
| loadbalancer_port | Port exposed by pulp-web service when ingress_type==loadbalancer | int32 | false |

[Back to Custom Resources](#custom-resources)

#### PulpStatus

PulpStatus defines the observed state of Pulp

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| conditions |  | []metav1.Condition | true |
| deployment_type | Name of the deployment type. | string | false |
| object_storage_azure_secret | The secret for Azure compliant object storage configuration. | string | false |
| object_storage_s3_secret | The secret for S3 compliant object storage configuration. | string | false |
| db_fields_encryption_secret | Secret where the Fernet symmetric encryption key is stored. | string | false |
| ingress_type | The ingress type to use to reach the deployed instance | string | false |
| ingress_class_name | IngressClassName is used to inform the operator which ingressclass should be used to provision the ingress. | string | false |
| container_token_secret | Secret where the container token certificates are stored. | string | false |
| admin_password_secret | Secret where the administrator password can be found | string | false |
| external_cache_secret | Name of the secret with the parameters to connect to an external Redis cluster | string | false |
| storagePersistentVolumeClaim | [DEPRECATED] Temporarily adding to keep compatibility with ansible version. | string | false |
| webURL |  | string | false |
| databaseConfigurationSecret |  | string | false |
| storageType |  | string | false |
| storageSecret |  | string | false |
| deployedVersion |  | string | false |
| deployedImage |  | string | false |
| migrantDatabaseConfigurationSecret |  | string | false |
| dbFieldsEncryptionSecret |  | string | false |
| upgradedPostgresVersion |  | string | false |
| migration_done |  | bool | false |

[Back to Custom Resources](#custom-resources)

#### Redis

[DEPRECATED] Temporarily adding to keep compatibility with ansible version

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| log_level | [TODO] Implement this like in ansible version [DEPRECATED?] Temporarily adding to keep compatibility with ansible version. | string | false |
| replicas | [TODO] THIS SHOULD BE REMOVED SINCE WE WILL NOT SUPPORT REDIS CLUSTER Keeping it just as a matter of compatibility | int32 | true |
| redis_resource_requirements |  | *corev1.ResourceRequirements | false |
| resource_requirements |  | *corev1.ResourceRequirements | false |
| strategy |  | *appsv1.DeploymentStrategy | false |

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

[Back to Custom Resources](#custom-resources)

#### Worker

Worker defines desired state of pulpcore-worker resources

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-worker replicas. Default: 2 | int32 | true |
| resource_requirements | Resource requirements for the pulp-api container | corev1.ResourceRequirements | false |
| affinity | Affinity is a group of affinity scheduling rules. | *corev1.Affinity | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |

[Back to Custom Resources](#custom-resources)
