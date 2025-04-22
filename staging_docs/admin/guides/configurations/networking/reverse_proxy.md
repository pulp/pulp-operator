# Reverse Proxy

[Pulp’s architecture](https://pulpproject.org/pulpcore/docs/admin/learn/architecture/) makes use of a reverse proxy that sits in front of the REST API and the content serving application:
![Architecture](https://pulpproject.org/pulpcore/docs/assets/images/architecture.png "Pulp’s architecture")

A pulp plugin may have [webserver snippets](https://pulpproject.org/pulpcore/docs/dev/learn/plugin-concepts/#configuring-reverse-proxy-with-custom-urls) to route custom URLs.

The operator convert these snippets into paths on [routes](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/routes/) and NGINX Ingress Controllers.

For now, because of a limitation (they do not support `rewrite rules` in their load balancer) in [`AWS`](https://github.com/kubernetes-sigs/aws-load-balancer-controller/issues/835) and [`GCE`](https://github.com/kubernetes/ingress-gce/issues/109) ingress controllers ([controllers supported and maintained by Kubernetes project](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/)), Pulp operator will keep deploying `pulp-web` and `Ingresses` for "*non-nginx*" controllers.

<br/>

# Manually Configuring Ingress Resources

There are cases in which the `Ingress` resource provided by Pulp Operator does not meet the usage requirements or it is required to have custom configurations not available through Pulp CR.

For such situations, it is possible to configure it manually.

The following `yaml` file is a template with all the *snippets* used by Pulp: [ingress.yaml](ingress.yaml).  
It can be used as an example to configure the `Ingress` *backend rules*.

After modifying the file with the expected configurations, create the `Ingress` resource:
```
$ kubectl apply -f ingress.yaml
```

and update Pulp CR with the `hostname` used in `Ingress`:

* create/update the `ConfigMap` (used by [`custom_pulp_settings`](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/pulp_settings/#custom-settings)):
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
data:
  token_server: '"http://<ingress_host>/token/"'
  content_origin: '"http://<ingress_host>"'
  ansible_api_hostname: '"http://<ingress_host>"'
  pypi_api_hostname: '"http://<ingress_host>"'
```

* update `Pulp CR` with the CM:
```yaml
spec:
  custom_pulp_settings: settings
```

!!! note
    Resources manually created will **not** be managed by the operator, which means,
    the operator will not reconcile or verify if this resource has the necessary configurations for
    Pulp's proper execution.
