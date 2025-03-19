# Installing Pulp Operator with Helm

!!! warning
    This installation method is still under development!

### *Prerequisites*
* [Helm cli](https://helm.sh/docs/intro/install/) installed


### Installing

* Add `pulp-operator` Helm repository:
```
helm repo add pulp-operator https://github.com/pulp/pulp-k8s-resources/raw/main/helm-charts/ --force-update
```

* [**optional**] Create a namespace to run `pulp-operator`:
```
kubectl create ns pulp
kubectl config set-context --current --namespace pulp
```

* Install `pulp-operator`:
```
helm -n pulp install pulp pulp-operator/pulp-operator
```


### Deploying Pulp

After installing `pulp-operator` we need to create a `Pulp CR` with the configurations to deploy `Pulp`.  
For example:
```
$ oc apply -f- <<EOF
apiVersion: repo-manager.pulpproject.org/v1
kind: Pulp
metadata:
  name: pulp
  namespace: pulp
spec:
  database:
    postgres_storage_class: standard

  file_storage_access_mode: "ReadWriteOnce"
  file_storage_size: "2Gi"
  file_storage_storage_class: standard
EOF
```

See [Custom Resources](/pulp_operator/pulp/) for more information about the available fields of `Pulp CR` or check our [list of samples](https://github.com/pulp/pulp-operator/tree/main/config/samples).
