# Settings


Pulp uses dynaconf for its settings which allows you to configure Pulp settings
through a configuration file (`/etc/pulp/settings.py`) that is automatically
created by Pulp Operator.

Check pulpcore doc for more information about the list of settings: [Pulp Settings](https://docs.pulpproject.org/pulpcore/configuration/settings.html).

## View Settings

To list the effective settings on a Pulp installation, run the command `dynaconf list`
from a Pulp API pod:

```sh
$ kubectl exec $(kubectl get deployment -oname -l app.kubernetes.io/component=api) -- dynaconf list
```

To check the `settings.py` file:
```sh
$ kubectl exec $(kubectl get deployment -oname -l app.kubernetes.io/component=api) -- cat /etc/pulp/settings.py
```

## Pulp Server Secret

To share the settings between Pulp pods, Pulp Operator creates a
[Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/)
(the [`pulp-server`](/pulp_operator/configuring/secrets/#pulp-server) Secret)
based on the definitions of `Pulp CR`.

There are 2 ways to configure the settings:

* [via specific fields](#pulp-operator-defined-settings)
* [via `custom_pulp_settings` field](#custom-settings)

## Pulp Operator Defined Settings

The following settings (database, cache, secret_key, etc) are all
"abstracted" from `Pulp CR` definitions and, under the hood, the operator
translates/migrates these configs into `settings.py`. To modify them, modify the
corresponding field or resource.

### Database

If `database.external_db_secret` is defined, Pulp Operator will configure the `settings.py`
file with the values from the Secret. If not, it will use the configs from the
self-managed database.
```python
DATABASES = {
  "default": {
    "HOST": ...,
    "ENGINE": ...,
    "NAME": ...,
    "USER": ...,
    "PASSWORD": ...,
    "PORT": ...,
    "CONN_MAX_AGE": 0,
    "OPTIONS": { "sslmode": ... },
  }
}
```

Check [Configuring Pulp Database](/pulp_operator/configuring/database/) for more
information on how to configure Pulp database.

### Cache

If `cache.enabled: true`, Pulp Operator will define the `REDIS_*` settings with
the definitions from `cache.external_cache_secret` Secret or from the self-managed
redis instance.
```python
CACHE_ENABLED = True
REDIS_HOST =  ...
REDIS_PORT =  ...
REDIS_PASSWORD = ...
REDIS_DB = ...
```

Check [Configuring Pulp Cache](/pulp_operator/configuring/cache/) for more
information on how to configure Pulp cache.

### Object Storage

If `object_storage_azure_secret` is defined, Pulp Operator will define the following
fields with the Secret's content:
```python
STORAGES = {
    "default": {
        "BACKEND": "storages.backends.azure_storage.AzureStorage",
        "OPTIONS": {
            "connection_string": ...,
            "location": ...,
            "account_name": ...,
            "azure_container": ...,
            "account_key": ...,
            "expiration_secs": 60,
            "overwrite_files": True,
        },
    },
}
MEDIA_ROOT = ""
```

If `object_storage_s3_secret` is defined, Pulp Operator will define the following
fields with the Secret's content:
```python
STORAGES = {
    "default": {
        "BACKEND": "storages.backends.s3boto3.S3Boto3Storage",
        "OPTIONS": {
            "bucket_name": ...,
            "access_key": ...,
            "secret_key": ...,
            "region_name": ...,
            "endpoint_url": ...,
            "signature_version": "s3v4",
            "addressing_style": ...,
        },
    },
}
MEDIA_ROOT = ""
```

Check [Configuring Pulp storage configuration](/pulp_operator/configuring/storage/)
for more information on how to configure Pulp storage.

### Fields that depend on `ingress_type`

Some fields are defined based on the `ingress_type`:
```python
ANSIBLE_API_HOSTNAME = ...
CONTENT_ORIGIN = ...
TOKEN_SERVER = ...
```

* if `ingress_type: ingress` the operator will set these fields with `ingress_host` value
* if `ingress_type: route` it will use the `route_host` definition
* if `ingress_type: ""` it will use the hostname from
    * `pulp-api` Service for the `TOKEN_SERVER`
    * `pulp-web` Service for the others


Check [Ingress](/pulp_operator/configuring/networking/exposing/#ingress) for more
information on how to expose Pulp to outside of k8s cluster.

### Secret Key

If `pulp_secret_key` is defined in Pulp CR, Pulp Operator will define the `SECRET_KEY`
in `settings.py` with it. <br/>
If `pulp_secret_key` is not defined, Pulp Operator will generate a random key and
configure `SECRET_KEY` with it.

Check [pulp-secret-key](/pulp_operator/configuring/secrets/#pulp-secret-key)
for more information about Django Secret Key.

### Allowed Checksum

If `allowed_content_checksums` is defined in Pulp CR, Pulp Operator will define
the `ALLOWED_CONTENT_CHECKSUMS` in `settings.py` with it. <br/>
If `allowed_content_checksums` is not defined, the `ALLOWED_CONTENT_CHECKSUMS`
setting will not be added to `settings.py` file.

Check [Configuring Pulp Allowed Content Checksums](/pulp_operator/configuring/content_checksums)
for more information about Pulp allowed checksum algorithms.

### LDAP

If `ldap.config` is defined in Pulp CR, Pulp Operator will do the following
configurations in `settings.py`:

* update the `AUTHENTICATION_BACKENDS`
```
AUTHENTICATION_BACKENDS = [
  "django_auth_ldap.backend.LDAPBackend",
  "django.contrib.auth.backends.ModelBackend",
  "pulpcore.backends.ObjectRolePermissionBackend",
]
```

* set the `AUTH_LDAP_*` fields with the "*converted*" (Pulp Operator will change
all Secret keys to uppercase and parse their values from YAML to a format
accepted by Python) values from the Secret defined in `ldap.config`.

Check [LDAP AUTHENTICATION](/pulp_operator/configuring/ldap) for more
information on how to configure Pulp to authenticate using LDAP.

### Default Settings

These fields are defined with default values.
```python
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys/"
PRIVATE_KEY_PATH = "/etc/pulp/keys/container_auth_private_key.pem"
PUBLIC_KEY_PATH = "/etc/pulp/keys/container_auth_public_key.pem"
STATIC_ROOT = "/var/lib/operator/static/"
TOKEN_AUTH_DISABLED = False
TOKEN_SIGNATURE_ALGORITHM = "ES256"
API_ROOT = "/pulp/"
```


## Custom Settings

!!! WARNING
    Use the `custom_pulp_settings` field with caution. Since Pulp Operator will not manage
    nor validate the contents from the ConfigMap, providing invalid values can cause disruption or
    unexpected behaviors.

Most of Pulp configurations should be done using the settings [presented before](/pulp_operator/configuring/pulp_settings/#pulp-operator-defined-settings),
but sometimes it is not possible. In this case, Pulp CR has the `custom_pulp_settings`
field that can be used to define a `ConfigMap` with the additional Pulp configurations.

For example, to disable
[Pulp analytics](https://docs.pulpproject.org/pulpcore/configuration/settings.html#analytics), first create a new ConfigMap:
```bash
$ kubectl create configmap settings  --from-literal=ANALYTICS=False
```

update Pulp CR with this new `ConfigMap`:

```yaml
spec:
  custom_pulp_settings: settings
```


!!! Info
    The `pulp_settings` field is deprecated!
    Use the `custom_pulp_settings` field instead.
