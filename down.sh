#!/bin/bash

# down.sh: Delete pulp-operator from K8s

# Remove the containers/pods before everything they depend on.
kubectl delete -f deploy/operator.yaml

kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/role_binding.yaml
# It doesn't matter which cr we specify; the metadata up top is the same.
kubectl delete -f deploy/crds/pulpproject_v1alpha1_pulp_cr.default.yaml
kubectl delete -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
