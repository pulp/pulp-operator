#!/bin/bash

# up.sh: Deploy pulp-operator to K8s

# CentOS 7 /etc/sudoers does not include /usr/local/bin
# Which k3s installs to.
# But we do not want to break other possible kubectl implementations by
# hardcoding /usr/local/bin/kubectl .
# And this entire script may be run with sudo.
# So if kubectl is not in the PATH, assume /usr/local/bin/kubectl .
set -x
if ! command -v kubectl > /dev/null; then
  if [ -x /usr/local/bin/kubectl ]; then
    echo "$0: kubectl not found, but /usr/local/bin/kubectl was found."
    echo "    Setting kubectl temporariily as an alias to /usr/local/bin/kubectl ."
    alias kubectl=/usr/local/bin/kubectl
    shopt -s expand_aliases
  else
    echo "$0: ERROR 1: Cannot find kubectl"
  fi
fi
# TODO: Check if all of these are needed.
# TODO: Check if these should only ever be run once; or require
# special logic to update
# The Custom Resource (and any ConfigMaps) do not.
kubectl apply -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
if [[ -e deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml ]]; then
  CUSTOM_RESOURCE=pulpproject_v1alpha1_pulp_cr.yaml
elif [[ "$TRAVIS" == "true" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1alpha1_pulp_cr.travis.yaml
elif [[ "$(hostname)" == "pulp-demo"* ]]; then
  CUSTOM_RESOURCE=pulpproject_v1alpha1_pulp_cr.pulp-demo.yaml
else
  CUSTOM_RESOURCE=pulpproject_v1alpha1_pulp_cr.default.yaml
fi
echo "Will deploy config Custom Resource deploy/crds/$CUSTOM_RESOURCE"
kubectl apply -f deploy/crds/$CUSTOM_RESOURCE
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/operator.yaml
