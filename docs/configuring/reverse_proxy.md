# Reverse Proxy

[Pulp’s architecture](https://docs.pulpproject.org/pulpcore/components.html) makes use of a reverse proxy that sits in front of the REST API and the content serving application:
![Architecture](https://docs.pulpproject.org/pulpcore/_images/architecture.png "Pulp’s architecture")

A pulp plugin may have [webserver snippets](https://docs.pulpproject.org/pulpcore/plugins/plugin-writer/concepts/index.html#configuring-reverse-proxy-with-custom-urls) to route custom URLs.

The operator convert these snippets into paths on [routes](https://docs.pulpproject.org/pulp_operator/configuring/routes/) and NGINX Ingress Controllers.

For now, because of a limitation (they do not support `rewrite rules` in their load balancer) in [`AWS`](https://github.com/kubernetes-sigs/aws-load-balancer-controller/issues/835) and [`GCE`](https://github.com/kubernetes/ingress-gce/issues/109) ingress controllers ([controllers supported and maintained by Kubernetes project](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/)), Pulp operator will keep deploying `pulp-web` and `Ingresses` for "*non-nginx*" controllers.
