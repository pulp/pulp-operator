# Pulp Operator log level configuration

It is possible to increase the log level of Pulp operator for troubleshooting purposes or reduce the log level for storage constraints, for example.

To do so, modify the `--zap-log-level=<new-level>` ARG from manager container of operator controller-manager deployment, where the `<new-level>` can be one of:

* debug
* info
* error


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
        - "--zap-stacktrace-level=error"
```
