# Reverse Proxy

[Pulp’s architecture](https://docs.pulpproject.org/pulpcore/components.html) makes use of a reverse proxy that sits in front of the REST API and the content serving application:
![Architecture](https://docs.pulpproject.org/pulpcore/_images/architecture.png "Pulp’s architecture")

A pulp plugin may have [webserver snippets](https://docs.pulpproject.org/pulpcore/plugins/plugin-writer/concepts/index.html#configuring-reverse-proxy-with-custom-urls) to route custom URLs.

The operator convert these snippets into paths on [routes](https://docs.pulpproject.org/pulp_operator/configuring/routes/) and NGINX Ingress Controllers, for other contollers, we provide the web service with the [web image](https://docs.pulpproject.org/pulp_operator/container/#pulp-web).
