#!/bin/bash

# up.sh: Deploy pulp-operator to K8s

# TODO: Check if all of these are needed.
# TODO: Check if these should only ever be run once; or require
# special logic to update
kubectl apply -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
kubectl apply -f deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml

if [[ -e deploy/pulp-operator.config-map.yml ]]; then
  CONFIG_MAP=pulp-operator.config-map.yml
elif [[ "$TRAVIS" == "true" ]]; then
  CONFIG_MAP=pulp-operator.travis.config-map.yml
elif [[ "$(hostname)" == "pulp-demo"* ]]; then
  CONFIG_MAP=pulp-operator.pulp-demo.config-map.yml
else
  CONFIG_MAP=pulp-operator.default.config-map.yml
fi
echo "Will deploy ConfigMap $CONFIG_MAP"
kubectl apply -f deploy/$CONFIG_MAP
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/operator.yaml
