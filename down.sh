#!/bin/bash

# down.sh: Delete pulp-operator from K8s

# Remove the containers/pods before everything they depend on.
kubectl delete -f deploy/operator.yaml

kubectl delete -f deploy/pulp-operator.default.config-map.yml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/role_binding.yaml

# TODO: Check if all of these are needed.
# TODO: Check if these should only ever be run once; or require
# special logic to update
kubectl delete -f deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml
kubectl delete -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
