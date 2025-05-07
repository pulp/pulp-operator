# Install Pulp Operator in OpenShift environments

Cluster administrators can use `OperatorGroups` to allow regular users to install Operators.  
To do so, as a `cluster-admin`, create an `OperatorGroup` in the namespace where regular
users would be able to install Pulp. For example:
```
$ oc apply -f- <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: pulp-operator-group
  namespace: pulp
spec:
  targetNamespaces:
  - pulp
EOF
```

See OpenShift official documentation for more information: [Operator Groups](https://docs.openshift.com/container-platform/latest/operators/understanding/olm/olm-understanding-operatorgroups.html#olm-understanding-operatorgroups)


### Install Pulp Operator as a regular user

If the `OperatorGroup` is already present in the namespace, a user with `edit` or `admin` role will be able to install Pulp operator:
```
$ oc apply -f- <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  labels:
    operators.coreos.com/pulp-operator.pulp: ""
  name: pulp-operator
  namespace: pulp
spec:
    channel: beta
    installPlanApproval: Automatic
    name: pulp-operator
    source: community-operators
    sourceNamespace: openshift-marketplace
    startingCSV: pulp-operator.v1.0.0-alpha.4
EOF
```

!!! note

    Role-based access control (RBAC) for Subscription objects is automatically granted to every user with the edit or admin role in a namespace. However, RBAC does not exist on OperatorGroup objects; this absence is what prevents regular users from installing Operators. Pre-installing Operator groups is effectively what gives installation privileges.  
    See OpenShift official documentation for more information: [Understanding Operator installation policy](https://docs.openshift.com/container-platform/latest/operators/admin/olm-creating-policy.html#olm-policy-understanding_olm-creating-policy)



### Deploy Pulp Operator

After configuring the `Subscription` the only remaining step is to configure `Pulp CR`.
For example:
```
$ oc apply -f- <<EOF
apiVersion: repo-manager.pulpproject.org/v1beta2
kind: Pulp
metadata:
  name: pulp
  namespace: pulp
spec:
  api:
    replicas: 1
  content:
    replicas: 1
  worker:
    replicas: 1
EOF
```

See [Custom Resources](/pulp_operator/pulp/) for more information about the available fields of `Pulp CR` or check our [list of samples](https://github.com/pulp/pulp-operator/tree/main/config/samples).


## Install multiple instances of Pulp

To deploy multiple instances of Pulp in different namespaces, repeat the following steps in each namespace where Pulp should be installed:

* (as a `cluster-admin`) create an [`OperatorGroup`](/pulp_operator/install/install/)
* (as a `regular user`) create a [`Subscription`](#installing-pulp-operator-as-a-regular-user)
* (as a `regular-user`) [deploy the operator](#deploying-pulp-operator)
