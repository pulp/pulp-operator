#!/bin/bash -e
#!/usr/bin/env bash

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

if [[ -z "${QUAY_EXPIRE+x}" ]]; then
  echo "Deploy pulp-operator"
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  export QUAY_IMAGE_TAG=$(cat Makefile | grep "VERSION ?=" | cut -d' ' -f3)

  echo $QUAY_IMAGE_TAG
  docker tag quay.io/pulp/pulp-operator:latest quay.io/pulp/pulp-operator:$QUAY_IMAGE_TAG
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  make bundle-build
  sudo -E QUAY_REPO_NAME=pulp-operator-bundle $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh


  wget https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest-4.7/opm-linux.tar.gz
  tar xvf opm-linux.tar.gz
  sudo mv opm /usr/local/bin/opm
  sudo chmod +x /usr/local/bin/opm


  opm index add -c docker --bundles quay.io/pulp/pulp-operator-bundle:${QUAY_IMAGE_TAG} --tag quay.io/pulp/pulp-index:${QUAY_IMAGE_TAG}
  sudo -E QUAY_REPO_NAME=pulp-index $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  docker images
fi

docker images
