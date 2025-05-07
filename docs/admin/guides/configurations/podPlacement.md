# Advanced pod scheduling

Pulp operator allows to restrict the nodes in which its pod should run through [nodeSelectors](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) and [node affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity).

!!! info

    If both nodeSelector and nodeAffinity are specified, both must be satisfied for the Pods to be scheduled onto a node.

For more information on how k8s pod assignment to nodes work, please consult the [official k8s documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/).


To define `nodeSelectors` to the pods deployed by Pulp operator configure the following fields:

* `api.node_selector` [**optional**] k8s will schedule api pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.
* `content.node_selector` [**optional**] k8s will schedule content pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.
* `worker.node_selector` [**optional**] k8s will schedule worker pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.
* `web.node_selector` [**optional**] k8s will schedule web pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.
* `cache.node_selector` [**optional**] k8s will schedule cache pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.
* `database.node_selector` [**optional**] k8s will schedule database pods onto nodes that have each of the labels specified. If not defined the k8s scheduler will not use nodeSelector to determine pod placement.


To define `node affinity` for Pulp operator pods:

* `api.affinity` [**optional**] specifies node affinities (`.spec.affinity.nodeAffinity`) field for api pods. If not defined the k8s scheduler will not use `node affinity` to determine pod placement.
* `content.affinity` [**optional**] specifies node affinities (`.spec.affinity.nodeAffinity`) field for content pods. If not defined the k8s scheduler will not use `node affinity` to determine pod placement.
* `worker.affinity`  [**optional**] specifies node affinities (`.spec.affinity.nodeAffinity`) field for worker pods. If not defined the k8s scheduler will not use `node affinity` to determine pod placement.
* `database.affinity` [**optional**] specifies node affinities (`.spec.affinity.nodeAffinity`) field for database pods. If not defined the k8s scheduler will not use `node affinity` to determine pod placement.