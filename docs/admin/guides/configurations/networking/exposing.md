# Expose Pulp to outside of Kubernetes cluster


To configure how Pulp will be externally accessible it is possible to define the `ingress_type` with
the following options:

* `nodeport`: expose Pulp resources through a k8s `NodePort` `Service`
* `ingress`: expose Pulp resources using k8s `Ingress`
* `route`: expose Pulp resources by creating OCP `Routes` (available only in OpenShift clusters)
* `loadbalancer`: expose Pulp resources through a k8s `LoadBalancer` `Service`

Only a single definition of `ingress_type` is allowed, which means, if Pulp CR is
configured with `ingress_type: nodeport` it is not possible to also define Pulp operator
to deploy `route` or `loadbalancer` resources for example.

If the configuration deployed by Pulp operator does not meet the required architecture
it is possible to manually provision the desired resources following the samples
provided in this documentation. For example, if it is required to deploy `route`
resources to allow external access to Pulp, but also `pulp-web` to serve as a reverse proxy
for internal applications communication, the operator can be configured with
`ingress_type: route` and the `pulp-web` resources should be manually provisioned.


# NodePort

The `nodeport` type will create `pulp-web` load balancers that will redirect the
traffic of `pulpcore-api` and `pulpcore-content` pods. `pulp-web` `Service` will be
exposed on one port of every node.

It is possible to define the port that will be exposed, but if no one is defined
a random port from k8s `service-node-port-range` definition will be used.

Example of `nodeport` configuration:
```
spec:
  ingress_type: nodeport
  nodeport_port: 30001
```

For more information on what is a k8s `Service` type `LoadBalancer` check the [Kubernetes project documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport).


# Ingress

Defining `ingress_type: ingress` will create `Ingress` resources to Pulp endpoints.
Since the k8s [`Ingressess`](https://kubernetes.io/docs/concepts/services-networking/ingress/) will redirect the traffic to pulpcore components, there
will be no need to provision `pulp-web` objects.

More information on configuring Pulp operator with `Ingress` can be found in [Reverse Proxy section](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/reverse_proxy/) .


# Route


!!! note
    `Routes` are resources available only in OpenShift clusters.

Defining `ingress_type: route` will create `Route` resources to Pulp endpoints.
Since OCP [`Routes`](https://docs.openshift.com/container-platform/4.13/networking/routes/route-configuration.html) will redirect the traffic to pulpcore components, there
will be no need to provision `pulp-web` objects.

More information on configuring Pulp operator with `Routes` can be found in [Routes section](https://pulpproject.org/pulp-operator/docs/admin/guides/configurations/networking/routes/).


# LoadBalancer

The `loadbalancer` type will create `pulp-web` load balancers that will redirect the
traffic of `pulpcore-api` and `pulpcore-content` pods. `pulp-web` will be exposed
by an external loadbalancer (if the cloud provider supports it).

Example of `loadbalancer` configuration:
```
spec:
  ingress_type: loadbalancer
```

For more information on what is a k8s `Service` type `LoadBalancer` check the [Kubernetes project documentation](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer).
