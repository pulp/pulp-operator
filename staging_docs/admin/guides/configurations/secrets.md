# Pulp Operator Secrets

Pulp Operator creates k8s `Secrets` based on the configuration defined in Pulp `CR`.

Some `Secrets` are **not** reconciled, which means, any modification in their content will **not**
get synchronized with the `CR` definition. This is to avoid losing any custom data added to the `Secret`.


### Restore the default values

To restore the default values defined by the Operator it is possible to remove the secret and let the Operator recreate it:

!!! warning
    This is a disruptive action. Any change made directly into the `Secret` will be lost.  
    We recommend to make a backup of the `Secret` before removing it.

* make a copy of the `Secret`
```
$ kubectl get secret -oyaml > my_secret.yaml
```

* delete the secret (the Operator will create a new one with the default values)
```
$ kubectl delete secret <secret name>
```

!!! note
    Any modifications to the `Secrets` [will **not** be replicated to the running pods](/pulp_operator/faq/#i-modified-a-configmapsecret-but-the-new-config-is-not-replicated-to-pods).  
    To update the `Pods` with the new `Secret` contents just delete the `Pod` (the new `Pod` provisioned by the controller will mount the updated `Secret`).

## List of Secrets deployed by the Operator

The following `Secrets` are created by the operator in case they are not provided through Pulp `CR`.
The name of the `Secrets` can be different depending on the Pulp's `CR` name.  

!!! note
    For the sake of simplicity, we are considering that the Operator `.metadata.name` is "*pulp*",
    so all of the following `Secrets` will be prefixed with `pulp-`.


### pulp-server

Will be used to populate `/etc/pulp/settings.py` configuration file.  

!!! warning
    Do not modify this `Secret`, the content will get overwritten by the operator.
    Any modification in `Pulp CR` that impact changing the content of this
    `Secret` will trigger a redeploy of `pulp-api` and `pulp-content` pods.

Here is an example of a `Secret` created by the Operator:

```
DB_ENCRYPTION_KEY = "/etc/pulp/keys/database_fields.symmetric.key"
GALAXY_COLLECTION_SIGNING_SERVICE = "ansible-default"
GALAXY_CONTAINER_SIGNING_SERVICE = "container-default"
ANSIBLE_API_HOSTNAME = "http://pulp-web-svc.pulp.svc.cluster.local:24880"
ANSIBLE_CERTS_DIR = "/etc/pulp/keys/"
CONTENT_ORIGIN = "http://pulp-web-svc.pulp.svc.cluster.local:24880"
DATABASES = {
        'default': {
                'HOST': 'postgres.db.svc.cluster.local',
                'ENGINE': 'django.db.backends.postgresql_psycopg2',
                'NAME': 'pulp',
                'USER': 'pulp-admin',
                'PASSWORD': 'password',
                'PORT': '5432',
                'CONN_MAX_AGE': 0,
                'OPTIONS': { 'sslmode': 'prefer' },
        }
}
GALAXY_FEATURE_FLAGS = {
        'execution_environments': 'True',
}
PRIVATE_KEY_PATH = "/etc/pulp/keys/container_auth_private_key.pem"
PUBLIC_KEY_PATH = "/etc/pulp/keys/container_auth_public_key.pem"
STATIC_ROOT = "/var/lib/operator/static/"
TOKEN_AUTH_DISABLED = "False"
TOKEN_SERVER = "http://pulp-api-svc.pulp.svc.cluster.local:24817/token/"
TOKEN_SIGNATURE_ALGORITHM = "ES256"
API_ROOT = "/pulp/"
CACHE_ENABLED = "True"
REDIS_HOST =  "pulp-redis-svc.pulp"
REDIS_PORT =  "6379"
REDIS_PASSWORD = ""
REDIS_DB = ""

```

For more information about Pulp Settings config file see [Pulpcore doc](https://docs.pulpproject.org/pulpcore/configuration/settings.html). <br/>
For more information about how to configure `settings.py` file using Pulp
Operator see [Pulp Settings](/pulp_operator/configuring/pulp_settings/).


### pulp-db-fields-encryption

Symmetric key used to encrypt the data stored in the database.  
*The current version of Operator does not provide a way to modify this key yet.*


### pulp-admin-password

To define the password from Pulp admin user, create a `Secret` with a `password` key and set `admin_password_secret` with the name of the `Secret` created.

* in this example we are creating a secret called "*my-admin-password*" and the "*password*" key has "*MySuperSecretPassword*" as value
```
$ kubectl create secret generic my-admin-password --from-literal=password=MySuperSecretPassword
```
* now we need to set the `admin_password_secret` field  in the CR
```
...
spec:
  admin_password_secret: my-admin-password
...
```

If the `admin_password_secret` field is not defined with the name of a `Secret` the Operator will create one (called *pulp-admin-password*) with a random string.


### pulp-container-auth

Contains the keys which are going to be used for the [signing and validation of tokens](https://docs.pulpproject.org/pulp_container/authentication.html#token-authentication-label).  
It is managed by `container_token_secret` field in Pulp `CR`.

### pulp-secret-key

Name of the Kubernetes `Secret` with Django `SECRET_KEY`.  
From [Django doc](https://docs.djangoproject.com/en/4.2/ref/settings/#secret-key): "*A secret key for a particular Django installation. This is used to provide cryptographic signing, and should be set to a unique, unpredictable value.*"  
The `Secret.data.key` must be named **secret_key**.

* in this example we are creating a secret called "*my-django-secret-key*" and the "*secret_key*" key has "*MySuperSecretPassword*" as value
```bash
$ kubectl create secret generic my-django-secret-key --from-literal=secret_key=MySuperSecretPassword
```
* now we need to set the `pulp_secret_key` field  in the CR
```yaml
...
spec:
  pulp_secret_key: my-django-secret-key
...
```

If the `pulp_secret_key` field is not defined with the name of a `Secret` the Operator will create one (called *pulp-secret-key*) with a random string.  
