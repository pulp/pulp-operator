# Routes

OpenShift clusters use [routers](https://docs.openshift.com/container-platform/4.10/networking/ingress-operator.html#nw-ne-openshift-ingress_configuring-ingress) as the `Ingress Controller`.

Here are the fields used by Pulp operator to configure `routes` in OpenShift clusters:

* `ingress_type` must be defined as `route`, so that the operator knows that it needs to provision the `route paths`
* `route_host` [**optional**] this will be the hostname where Pulp can be accessed. If not defined, Pulp operator will define one based on default ingress domain name.
* `route_labels` [**optional**] a map of the labels that can be used by `routeSelector`. If not defined Pulp operator will create `routes` that will use the default `routers`.

For more information about `routeSelector` and `route sharding`, please consult the [official OpenShift documentation](https://docs.openshift.com/container-platform/4.10/networking/configuring_ingress_cluster_traffic/configuring-ingress-cluster-traffic-ingress-controller.html#nw-ingress-sharding-route-labels_configuring-ingress-cluster-traffic-ingress-controller).