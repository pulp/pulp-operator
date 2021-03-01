#!/bin/bash

# down.sh: Delete pulp-operator from K8s

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

# Remove the containers/pods before everything they depend on.
$KUBECTL delete -f deploy/operator.yaml

$KUBECTL delete -f deploy/service_account.yaml
$KUBECTL delete -f deploy/role.yaml
$KUBECTL delete -f deploy/cluster_role.yaml
$KUBECTL delete -f deploy/role_binding.yaml
$KUBECTL delete -f deploy/cluster_role_binding.yaml
# It doesn't matter which cr we specify; the metadata up top is the same.
$KUBECTL delete -f deploy/crds/pulpproject_v1beta1_pulp_cr.default.yaml
$KUBECTL delete -f deploy/crds/pulpproject_v1beta1_pulp_crd.yaml
