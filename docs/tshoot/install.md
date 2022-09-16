# Troubleshooting installations


In a successful installation, all pods should be in a `READY` state:
```bash
$ kubectl get pods
NAME                                    READY   STATUS    RESTARTS      AGE
pulp-api-59748b555f-wkqbb               1/1     Running   0             18h
pulp-content-545fb4d577-gdpcq           1/1     Running   0             18h
pulp-database-0                         1/1     Running   0             18h
pulp-redis-7c57966b76-kq4lf             1/1     Running   0             18h
pulp-web-548f9ff866-rwf9f               1/1     Running   0             18h
pulp-worker-75fb775dd7-5pgk7            1/1     Running   0             18h
```

Checking the operator logs, we should see the following message (indicating that there is no pending tasks):
```
$ kubectl logs deployment/pulp-operator-controller-manager
...
2022-09-16T13:53:28Z	INFO	pulp/controller.go:238	Operator tasks synced
...
```

From Pulp api pods we could also check cluster's health:
```json
$ kubectl exec deployment/example-pulp-api -- curl -s localhost:24817/pulp/api/v3/status/|jq
{
  "versions": [ ...
  ],
  "online_workers": [   <-------------- we should see the worker pods listed here
    {
      "pulp_href": "/pulp/api/v3/workers/70e84b43-5a31-431b-87d6-0ee1ea664355/",
      "pulp_created": "2022-09-16T12:52:22.053237Z",
      "name": "13@example-pulp-worker-75fb775dd7-5pgk7",
      "last_heartbeat": "2022-09-16T14:00:55.022812Z",
      "current_task": null
    }
  ],
  "online_content_apps": [     <-------------- we should see the content pods listed here
    {
      "name": "12@example-pulp-content-545fb4d577-gdpcq",
      "last_heartbeat": "2022-09-16T14:01:03.023915Z"
    }
  ],
  "database_connection": {
    "connected": true     <------- database_connection must be true
  },
  "redis_connection": {
    "connected": true      <------- redis_connection is optional (cache is not mandatory)
  },
  "storage": {
    "total": 32737570816,
    "used": 25801592832,
    "free": 6935977984
  }
}
```

Once a problem is identified and more help is needed, please follow the steps from *["Gathering data about Pulp installation"](gatherData.md)* documentation to share the installation data.