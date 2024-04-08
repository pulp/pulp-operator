# LOG LEVEL

## Pulp Operator log level configuration

It is possible to increase the log level of Pulp operator for troubleshooting purposes or reduce the log level for storage constraints, for example.

To do so, modify the `--zap-log-level=<new-level>` ARG from manager container of operator controller-manager deployment, where the `<new-level>` can be one of:

* debug
* info
* error

It is also possible to modify the `--zap-stacktrace-level` to:

* info
* error
* panic

```yaml
$ kubectl edit deployment/<deployment-name>-controller-manager
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <deployment-name>-controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
...
      - name: manager
        args:
        - "--health-probe-bind-address=:8081"
        - "--metrics-bind-address=127.0.0.1:8080"
        - "--leader-elect"
        - "--zap-log-level=debug"     <-------------------
        - "--zap-stacktrace-level=panic"  <---------------
```


## Pulpcore Pods Debug Logging

By default Pulp logs at INFO level, but enabling DEBUG logging can be a helpful thing to get more insight when things don’t go as expected.
This can be enabled by updating Pulp CR with:
```yaml
spec:
  enable_debugging: true
```
after that, the operator will update the *pulp-server* `Secret` with the expected `LOGGING` config and restart pulpcore pods to get the new configuration.

To disable it, remove the `enable_debugging` config or set it to false:
```yaml
spec:
  enable_debugging: false
```
