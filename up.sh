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

KUBE_ASSETS_DIR=${KUBE_ASSETS_DIR:-".ci/assets/kubernetes"}

# TODO: Check if these should only ever be run once; or require
# special logic to update
# The Custom Resource (and any ConfigMaps) do not.
$KUBECTL apply -f deploy/crds/pulpproject_v1beta1_pulp_crd.yaml
if [[ -e deploy/crds/pulpproject_v1beta1_pulp_cr.yaml ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.yaml
elif [[ "$CI_TEST" == "true" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.ci.yaml
  echo "Will deploy admin password secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-admin-password.secret.yaml
elif [[ "$CI_TEST" == "aws" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.object_storage.aws.yaml
  echo "Will deploy admin password secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-admin-password.secret.yaml
  echo "Will deploy object storage secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-object-storage.aws.secret.yaml
elif [[ "$CI_TEST" == "azure" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.object_storage.azure.yaml
  echo "Will deploy admin password secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-admin-password.secret.yaml
  echo "Will deploy object storage secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-object-storage.azure.secret.yaml
elif [[ "$CI_TEST" == "galaxy" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.galaxy.ci.yaml
  echo "Will deploy admin password secret for testing ..."
  $KUBECTL apply -f $KUBE_ASSETS_DIR/pulp-admin-password.secret.yaml
elif [[ "$(hostname)" == "pulp-demo"* ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.pulp-demo.yaml
else
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.default.yaml
fi
echo "Will deploy config Custom Resource deploy/crds/$CUSTOM_RESOURCE"
$KUBECTL apply -f deploy/crds/$CUSTOM_RESOURCE
$KUBECTL apply -f deploy/service_account.yaml
$KUBECTL apply -f deploy/role.yaml
$KUBECTL apply -f deploy/cluster_role.yaml
$KUBECTL apply -f deploy/role_binding.yaml
$KUBECTL apply -f deploy/cluster_role_binding.yaml
$KUBECTL apply -f deploy/operator.yaml
