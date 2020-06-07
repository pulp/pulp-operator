#!/bin/bash

# up.sh: Deploy pulp-operator to K8s

# CentOS 7 /etc/sudoers , and non-interactive shells (vagrant provisions)
# do not include /usr/local/bin , Which k3s installs to.
# But we do not want to break other possible kubectl implementations by
# hardcoding /usr/local/bin/kubectl .
# And this entire script may be run with sudo.
# So if kubectl is not in the PATH, assume /usr/local/bin/kubectl .
if command -v kubectl > /dev/null; then
  KUBECTL=$(command -v kubectl)
elif [ -x /usr/local/bin/kubectl ]; then
  KUBECTL=/usr/local/bin/kubectl
elif which kubectl > /dev/null; then
  KUBECTL=$(which kubectl)
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

# TODO: Check if these should only ever be run once; or require
# special logic to update
# The Custom Resource (and any ConfigMaps) do not.
$KUBECTL apply -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
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
$KUBECTL apply -f deploy/crds/$CUSTOM_RESOURCE
$KUBECTL apply -f deploy/service_account.yaml
$KUBECTL apply -f deploy/role.yaml
$KUBECTL apply -f deploy/cluster_role.yaml
$KUBECTL apply -f deploy/role_binding.yaml
$KUBECTL apply -f deploy/cluster_role_binding.yaml
$KUBECTL apply -f deploy/operator.yaml
