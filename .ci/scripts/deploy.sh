#!/bin/bash -e
#!/usr/bin/env bash

DEPLOY_FLAG=""
if
  [ "$1" = "--index" ] || [ "$1" = "-i" ]; then
  DEPLOY_FLAG="-i"
fi

if [[ "$DEPLOY_FLAG" == "-i" ]]; then
  echo "Deploy pulp-operator"
  eval $(minikube -p minikube docker-env)
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  export QUAY_IMAGE_TAG=$(python -c 'import yaml; print(yaml.safe_load(open("deploy/olm-catalog/pulp-operator/manifests/pulp-operator.clusterserviceversion.yaml"))["spec"]["version"])')
  sed -i "s/\.dev//g" deploy/olm-catalog/pulp-operator/manifests/pulp-operator.clusterserviceversion.yaml


  echo $QUAY_IMAGE_TAG
  docker tag quay.io/pulp/pulp-operator:latest quay.io/pulp/pulp-operator:$QUAY_IMAGE_TAG
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  docker build -f bundle.Dockerfile -t quay.io/pulp/pulp-operator-bundle:${QUAY_IMAGE_TAG} .
  sudo -E QUAY_REPO_NAME=pulp-operator-bundle $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh


  wget https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest-4.7/opm-linux.tar.gz
  tar xvf opm-linux.tar.gz
  sudo mv opm /usr/local/bin/opm
  sudo chmod +x /usr/local/bin/opm


  opm index add -c docker --bundles quay.io/pulp/pulp-operator-bundle:${QUAY_IMAGE_TAG} --tag quay.io/pulp/pulp-index:${QUAY_IMAGE_TAG}
  sudo -E QUAY_REPO_NAME=pulp-index $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  docker images
  exit
fi

if [[ "$CI_TEST" == "galaxy" ]]; then
  echo "Deploy galaxy latest"
  sudo -E QUAY_REPO_NAME=galaxy $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  echo "Deploy galaxy-web latest"
  sudo -E QUAY_REPO_NAME=galaxy-web $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
else
  echo "Deploy pulp latest"
  sudo -E QUAY_REPO_NAME=pulp $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

  echo "Deploy pulpcore latest"
  sudo -E QUAY_REPO_NAME=pulpcore $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

  echo "Deploy pulp-web latest"
  sudo -E QUAY_REPO_NAME=pulp-web $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
fi

docker images
