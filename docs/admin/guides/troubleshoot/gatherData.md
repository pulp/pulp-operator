# Gather data about Pulp installation

A practical way to collect information from Pulp operator is by running *[kubectl cluster-info](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#cluster-info)* or *[oc adm inspec](https://docs.openshift.com/container-platform/4.10/support/gathering-cluster-data.html)* commands.

They can gather outputs like deployment spec, pod logs, service spec, etc with a single command.

!!! note
    Make sure to obfuscate sensitive data when running `oc adm inspect` before sharing them.


## Gather data on Kubernetes clusters

* run `cluster-info`
~~~
$ PULP_NAMESPACE=pulp
$ kubectl cluster-info dump --namespaces=$PULP_NAMESPACE --output-directory=/tmp/cluster-info
$ kubectl -n $PULP_NAMESPACE get pulp -ojson > /tmp/cluster-info/pulp.json
~~~

* create a tar file to share the data
~~~
$ tar cvaf cluster-info.tar.gz /tmp/cluster-info/
~~~

## Gather data on OpenShift clusters

* run `oc adm inspect`
~~~
$ PULP_NAMESPACE=pulp
$ oc adm inspect ns/$PULP_NAMESPACE --dest-dir=/tmp/adm-inspect
$ oc -n $PULP_NAMESPACE get pulp -ojson > /tmp/adm-inspect/pulp.json
~~~

* remove sensitive information
~~~
$ rm /tmp/adm-inspect/namespaces/$PULP_NAMESPACE/core/secrets.yaml
~~~

* create a tar file to share the data
~~~
$ tar cvaf adm-inspect.tar.gz /tmp/adm-inspect/
~~~
