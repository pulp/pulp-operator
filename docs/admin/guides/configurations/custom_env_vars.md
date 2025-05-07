# Configure Custom Environment Variables
Pulp Operator provide the `env_vars` field to define custom environment variables for containers. <br/>

!!! INFO
    The following environment variables are managed by Pulp Operator and will be
    overwritten in case defined as custom env var:

    * **PULP_GUNICORN_TIMEOUT**
    * **PULP_API_WORKERS**
    * **PULP_CONTENT_WORKERS**
    * **REDIS_SERVICE_HOST**
    * **REDIS_SERVICE_PORT**
    * **REDIS_SERVICE_DB**
    * **REDIS_SERVICE_PASSWORD**
    * **PULP_SIGNING_KEY_FINGERPRINT**
    * **POSTGRES_SERVICE_HOST**
    * **POSTGRES_SERVICE_PORT**

For more information about Kubernetes environment variables, check the k8s official documentation:

* [Define Dependent Environment Variables](https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/)
* [Define Environment Variables for a Container](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
* [Define container environment variables using Secret data ](https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/#define-container-environment-variables-using-secret-data)


## API Deployments


Example of CR configuration for `pulpcore-api` containers:
```yaml
spec:
  api:
    env_vars:
    - name: "<env var name>"
      value: "<env var value>"
    - name: "<env var name>"
      valueFrom:
        secretKeyRef:
          key: <secret key>
          name: <secret name>
```

## Worker Deployments

Example of CR configuration for `pulpcore-worker` containers:
```yaml
spec:
  worker:
    env_vars:
    - name: "<env var name>"
      value: "<env var value>"
    - name: "<env var name>"
      valueFrom:
        secretKeyRef:
          key: <secret key>
          name: <secret name>
```

## Content Deployments

Example of CR configuration for `pulpcore-content` containers:
```yaml
spec:
  content:
    env_vars:
    - name: "<env var name>"
      value: "<env var value>"
    - name: "<env var name>"
      valueFrom:
        secretKeyRef:
          key: <secret key>
          name: <secret name>
```

## Web Deployments

Example of CR configuration for `pulpcore-web` containers:
```yaml
spec:
  web:
    env_vars:
    - name: "<env var name>"
      value: "<env var value>"
    - name: "<env var name>"
      valueFrom:
        secretKeyRef:
          key: <secret key>
          name: <secret name>
```

## Jobs

It is also possible to define custom env vars for the containers from `AdminPasswordJob`,
`MigrationJob`, and `SigningJob`. For example:
```yaml
spec:
  admin_password_job:
    container:
      env_vars:
      - name: "<env var name>"
        value: "<env var value>"
  migration_job:
    container:
      env_vars:
      - name: "<env var name>"
        value: "<env var value>"
  signing_job:
    container:
      env_vars:
      - name: "<env var name>"
        value: "<env var value>"
```
