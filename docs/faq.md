### **I modified a configmap/secret but the new config is not replicated to pods**

This is a known issue that [Pulp team is discussing](https://github.com/pulp/pulp-operator/issues/521) what will be the best way to handle it. The [Kubernetes community does not have a consensus](https://github.com/kubernetes/kubernetes/issues/22368) about it too.

One of the reasons that we don't want to put the pod restart logic (to propagate the new configuration) in operator is because it can cause downtime in case of error and we would need to automate the rollback or fix processes, which would probably bring other issues instead of being helpful.