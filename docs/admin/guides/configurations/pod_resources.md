# Setting Pod Resources

The pod resource requests and limits can be specified for each component.

Below is an example of a Pulp CR spec that sets CPU and memory resources.

It is not recommended to set the worker's CPU limit to less than 2.

Repository syncing takes longer when workers have insufficient CPU resources.

```yaml
  spec:
    api:
      replicas: 2
      gunicorn_workers: 1
      resource_requirements:
        requests:
          cpu: 250m
          memory: 256Mi
        limits:
          cpu: 1
          memory: 512Mi
    content:
      replicas: 2
      resource_requirements:
        requests:
          cpu: 250m
          memory: 256Mi
        limits:
          cpu: 500m
          memory: 512Mi
    worker:
      replicas: 5
      gunicorn_workers: 1
      resource_requirements:
        requests:
          cpu: 250m
          memory: 256Mi
        limits:
          cpu: 2
          memory: 5120Mi
    web:
      replicas: 1
      resource_requirements:
        requests:
          cpu: 250m
          memory: 128Mi
        limits:
          cpu: 500m
          memory: 128Mi
```