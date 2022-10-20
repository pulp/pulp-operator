
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
* [Web](#web)
* [Worker](#worker)

#### Affinity



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| nodeAffinity |  | *corev1.NodeAffinity | false |

[Back to Custom Resources](#custom-resources)

#### Api



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-api replicas. | int32 | true |
| affinity | Defines various deployment affinities. | [Affinity](#affinity) | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| gunicorn_timeout | The timeout for the gunicorn process. | int | false |
| gunicorn_workers | The number of gunicorn workers to use for the api. | int | false |
| resource_requirements | Resource requirements for the pulp api container. | corev1.ResourceRequirements | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |

[Back to Custom Resources](#custom-resources)

#### Cache



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| external_cache_secret | Name of the secret with the parameters to connect to an external Redis cluster | string | false |
| enabled | Defines if cache should be enabled. | bool | false |
| redis_image | The image name for the redis image. [default: \"redis:latest\"] | string | false |
| redis_storage_class | Storage class to use for the Redis PVC | string | false |
| redis_port | The port for Redis. [default: 6379] | int | false |
| redis_resource_requirements | Resource requirements for the Redis container | corev1.ResourceRequirements | false |
| pvc | PersistenVolumeClaim name that will be used by Redis pods If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| affinity | Defines various deployment affinities. | [Affinity](#affinity) | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |

[Back to Custom Resources](#custom-resources)

#### Content



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-content replicas | int32 | true |
| resource_requirements | Resource requirements for the pulp-content container | corev1.ResourceRequirements | false |
| affinity | Defines various deployment affinities. | [Affinity](#affinity) | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| gunicorn_timeout | The timeout for the gunicorn process. | int | false |
| gunicorn_workers | The number of gunicorn workers to use for the api. | int | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |

[Back to Custom Resources](#custom-resources)

#### Database



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| external_db_secret | Secret name with the configuration to use an external database | string | false |
| version | PostgreSQL version [default: \"13\"] | string | false |
| postgres_port | PostgreSQL port [default: 5432] | int | false |
| postgres_ssl_mode | Configure PostgreSQL connection sslmode option [default: \"prefer\"] | string | false |
| postgres_image | PostgreSQL container image [default: \"postgres:13\"] | string | false |
| postgres_extra_args |  | []string | false |
| postgres_data_path | Registry path to the PostgreSQL container to use [default: \"/var/lib/postgresql/data/pgdata\"] | string | false |
| postgres_initdb_args | Arguments to pass to PostgreSQL initdb command when creating a new cluster. [default: \"--auth-host=scram-sha-256\"] | string | false |
| postgres_host_auth_method | PostgreSQL host authentication method [default: \"scram-sha-256\"] | string | false |
| postgres_resource_requirements | Resource requirements for the database container. | corev1.ResourceRequirements | false |
| affinity | Defines various deployment affinities. | [Affinity](#affinity) | false |
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
| deployment_type | Name of the deployment type. | string | false |
| file_storage_size | The size of the file storage; for example 100Gi. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_access_mode | The file storage access mode. This field should be used only if file_storage_storage_class is provided | string | false |
| file_storage_storage_class | Storage class to use for the file persistentVolumeClaim | string | false |
| object_storage_azure_secret | The secret for Azure compliant object storage configuration. | string | false |
| object_storage_s3_secret | The secret for S3 compliant object storage configuration. | string | false |
| pvc | PersistenVolumeClaim name that will be used by Pulp pods If defined, the PVC must be provisioned by the user and the operator will only configure the deployment to use it | string | false |
| db_fields_encryption_secret | Secret where the Fernet symmetric encryption key is stored. | string | false |
| signing_secret | Secret where the signing certificates are stored. | string | false |
| signing_scripts_configmap | ConfigMap where the signing scripts are stored. | string | false |
| storage_type | Configuration for the storage type utilized in the backup | string | false |
| ingress_type | The ingress type to use to reach the deployed instance | string | false |
| ingress_annotations | Annotations for the Ingress | map[string]string | false |
| ingress_host | Ingress DNS host | string | false |
| route_host | Route DNS host | string | false |
| route_labels | RouteLabels will append custom label(s) into routes (used by router shard routeSelector). | map[string]string | false |
| nodeport_port | Provide requested port value | int32 | false |
| haproxy_timeout | The timeout for HAProxy. | string | false |
| nginx_client_max_body_size | The client max body size for Nginx Ingress | string | false |
| nginx_proxy_body_size | The proxy body size for Nginx Ingress | string | false |
| nginx_proxy_read_timeout | The proxy read timeout for Nginx Ingress | string | false |
| nginx_proxy_connect_timeout | The proxy connect timeout for Nginx Ingress | string | false |
| nginx_proxy_send_timeout | The proxy send timeout for Nginx Ingress | string | false |
| container_token_secret | Secret where the container token certificates are stored. | string | false |
| container_auth_public_key_name |  | string | false |
| container_auth_private_key_name |  | string | false |
| image | The image name (repo name) for the pulp image. | string | false |
| image_version | The image version for the pulp image. | string | false |
| image_pull_policy | Image pull policy for container image | string | false |
| api |  | [Api](#api) | false |
| database |  | [Database](#database) | false |
| content |  | [Content](#content) | false |
| worker |  | [Worker](#worker) | false |
| web |  | [Web](#web) | false |
| cache |  | [Cache](#cache) | false |
| pulp_settings | The pulp settings. | runtime.RawExtension | false |
| image_web | The image name (repo name) for the pulp webserver image. | string | false |
| image_web_version | The image version for the pulp webserver image. | string | false |
| admin_password_secret | Secret where the administrator password can be found | string | false |
| image_pull_secrets | Image pull secrets for container images | []string | false |
| sso_secret | Secret where Single Sign-on configuration can be found | string | false |
| mount_trusted_ca | Define if the operator should or should not mount the custom CA certificates added to the cluster via cluster-wide proxy config | bool | false |

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
| container_token_secret | Secret where the container token certificates are stored. | string | false |
| admin_password_secret | Secret where the administrator password can be found | string | false |
| external_cache_secret | Name of the secret with the parameters to connect to an external Redis cluster | string | false |

[Back to Custom Resources](#custom-resources)

#### Web



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-web replicas | int32 | true |
| resource_requirements | Resource requirements for the pulp-web container | corev1.ResourceRequirements | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| node_selector | NodeSelector for the Web pods. | map[string]string | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |

[Back to Custom Resources](#custom-resources)

#### Worker



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| replicas | Size is the size of number of pulp-worker replicas | int32 | true |
| resource_requirements | Resource requirements for the pulp-api container | corev1.ResourceRequirements | false |
| affinity | Defines various deployment affinities. | [Affinity](#affinity) | false |
| node_selector | NodeSelector for the Pulp pods. | map[string]string | false |
| tolerations | Node tolerations for the Pulp pods. | []corev1.Toleration | false |
| topology_spread_constraints | Topology rule(s) for the pods. | []corev1.TopologySpreadConstraint | false |
| readinessProbe | Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. | *corev1.Probe | false |
| livenessProbe | Periodic probe of container liveness. Container will be restarted if the probe fails. | *corev1.Probe | false |
| pdb | PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods | *policy.PodDisruptionBudgetSpec | false |
| strategy | The deployment strategy to use to replace existing pods with new ones. | appsv1.DeploymentStrategy | false |

[Back to Custom Resources](#custom-resources)
