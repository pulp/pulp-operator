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
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

echo "Make Install"
make install
echo "Make Deploy"
make deploy IMG=quay.io/pulp/pulp-operator:latest
echo "Namespaces"
$KUBECTL get namespace
echo "Set context"
$KUBECTL config set-context --current --namespace=pulp-operator-system
echo "Get Deployment"
$KUBECTL get deployment

# TODO: Check if these should only ever be run once; or require
# special logic to update
# The Custom Resource (and any ConfigMaps) do not.
if [[ -e config/samples/pulp_v1alpha1_pulp.yaml ]]; then
  CUSTOM_RESOURCE=pulp_v1alpha1_pulp.yaml
elif [[ "$CI_TEST" == "true" ]]; then
  CUSTOM_RESOURCE=pulp_v1alpha1_pulp.ci.yaml
elif [[ "$(hostname)" == "pulp-demo"* ]]; then
  CUSTOM_RESOURCE=pulp_v1alpha1_pulp.pulp-demo.yaml
else
  CUSTOM_RESOURCE=pulp_v1alpha1_pulp.default.yaml
fi
echo "Will deploy config Custom Resource config/samples/$CUSTOM_RESOURCE"
$KUBECTL apply -f config/samples/$CUSTOM_RESOURCE
