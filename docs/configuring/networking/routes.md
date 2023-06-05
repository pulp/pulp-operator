# Routes

OpenShift clusters use [`Routers`](https://docs.openshift.com/container-platform/4.10/networking/ingress-operator.html#nw-ne-openshift-ingress_configuring-ingress) as the `Ingress Controller`.

Here are the fields used by Pulp operator to configure `Routes` in OpenShift clusters:

* `ingress_type` must be defined as `route`, so that the operator knows that it needs to provision the `route paths`
* `route_host` [**optional**] this will be the hostname where Pulp can be accessed. If not defined, Pulp operator will define one based on default ingress domain name.
* `route_labels` [**optional**] a map of the labels that can be used by `routeSelector`. If not defined Pulp operator will create `Routes` that will use the default `Routers`.

For more information about `routeSelector` and `route sharding`, please consult the [official OpenShift documentation](https://docs.openshift.com/container-platform/4.10/networking/configuring_ingress_cluster_traffic/configuring-ingress-cluster-traffic-ingress-controller.html#nw-ingress-sharding-route-labels_configuring-ingress-cluster-traffic-ingress-controller).


## Configuring custom certificate

By default, Pulp Operator will provision `Routes` with [edge TLS termination](https://docs.openshift.com/container-platform/latest/networking/routes/secured-routes.html#nw-ingress-creating-an-edge-route-with-a-custom-certificate_secured-routes) (TLS encryption terminates on `Route`).

It is possible to configure the operator to deploy the `Routes` using a custom certificate.  
To do that, first create a `Secret` with the TLS certificate and key:
```
$ oc create secret generic <my-new-secret> --from-file=certificate=<cert file> --from-file=key=<key file>
```

For example:
```
$ oc create secret generic route-certs --from-file=certificate=/tmp/tls.crt --from-file=key=/tmp/tls.key
```

You may also specify a CA certificate if needed to complete the certificate chain:
```
$ oc create secret generic route-certs --from-file=certificate=/tmp/tls.crt --from-file=key=/tmp/tls.key --from-file=caCertificate=/tmp/ca.crt
```

!!! Warning
    Make sure to not modify the names of `Secrets`' keys: "***certificate***","***key***","***caCertificate***".  
    Using different key names will fail `Route` TLS config.

Now, configure Pulp CR with the `Secret` created:
```
...
spec:
  route_tls_secret: <my-new-secret>
...
```

A new reconciliation loop will be triggered and the certificate will be configured in all `Routes`.