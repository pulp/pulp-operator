# Configuring Pulp Cache

Pulp can cache the metadata instead of doing "slower" requests into the database (PostgreSQL).
To do so, it uses [Redis as the cache backend](https://docs.pulpproject.org/pulpcore/configuration/settings.html#redis-settings) technology.

Pulp operator provides a single node Redis server for Pulp to use, but it is also possible to configure the operator to use an external Redis installation.

## Configuring Pulp operator to deploy a Redis instance

[Pulp CR page](https://docs.pulpproject.org/pulp_operator/pulp/#cache) has all the parameters that can be set to inform Pulp operator how it should deploy the Redis container.

If no `cache` parameter is defined, Pulp operator will deploy Redis with the following configuration:

* a `Deployment` will be provisioned to handle Redis pod
* a single Redis replica will be available (it is **not** possible to form a cluster with this container)
* it will deploy a `docker.io/library/redis:latest` image

A `Service` will be created with the Redis pod as endpoint.

Here is an example of how to configure Pulp operator to deploy the Redis cache:
```
...
spec:
  cache:
    enabled: true
...
```

## Configuring Pulp operator to use an external Redis installation

It is also possible to configure Pulp operator to point to a running Redis cluster.
To do so, create a new `Secret` with the parameters to connect to the running Redis cluster:
```
$ kubectl -npulp create secret generic external-redis \
        --from-literal=REDIS_HOST=my-redis-host.example.com  \
        --from-literal=REDIS_PORT=6379  \
        --from-literal=REDIS_PASSWORD=""  \
        --from-literal=REDIS_DB=""
```

Make sure to define all the keys (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`) even if Redis cluster has
no authentication, like in the above example.

Now, configure Pulp operator CR to use the Secret:
```
...
spec:
  cache:
    enabled: true
    external_cache_secret: external-redis
...
```
