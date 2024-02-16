# Reverse Proxy

[Pulp’s architecture](https://docs.pulpproject.org/pulpcore/components.html) makes use of a reverse proxy that sits in front of the REST API and the content serving application:
![Architecture](https://docs.pulpproject.org/pulpcore/_images/architecture.png "Pulp’s architecture")

A pulp plugin may have [webserver snippets](https://docs.pulpproject.org/pulpcore/plugins/plugin-writer/concepts/index.html#configuring-reverse-proxy-with-custom-urls) to route custom URLs.

The operator convert these snippets into paths on [routes](https://docs.pulpproject.org/pulp_operator/configuring/routes/) and NGINX Ingress Controllers.

For now, because of a limitation (they do not support `rewrite rules` in their load balancer) in [`AWS`](https://github.com/kubernetes-sigs/aws-load-balancer-controller/issues/835) and [`GCE`](https://github.com/kubernetes/ingress-gce/issues/109) ingress controllers ([controllers supported and maintained by Kubernetes project](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/)), Pulp operator will keep deploying `pulp-web` and `Ingresses` for "*non-nginx*" controllers.

<br/>

# Manually Configuring Ingress Resources

There are cases in which the `Ingress` resource provided by Pulp Operator does not meet the usage requirements or it is required to have custom configurations not available through Pulp CR.

For such situations, it is possible to configure it manually.

The following `yaml` file is a template with all the *snipets* used by Pulp: [ingress.yaml](ingress.yaml).  
It can be used as an example to configure the `Ingress` *backend rules*.

After modifying the file with the expected configurations, create the `Ingress` resource:
```
$ kubectl apply -f ingress.yaml
```

and update Pulp CR with the `hostname` used in `Ingress`:
```yaml
spec:
  pulp_settings:
    content_origin: http://<ingress host>
    ansible_api_hostname: http://<ingress host>
    token_server: http://<ingress host>/token/
```

!!! note
    Resources manually created will **not** be managed by the operator, which means,
    the operator will not reconcile or verify if this resource has the necessary configurations for
    Pulp's proper execution.
