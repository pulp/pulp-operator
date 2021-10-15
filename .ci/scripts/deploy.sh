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
  make docker-push

  export QUAY_IMAGE_TAG=v$(cat Makefile | grep "VERSION ?=" | cut -d' ' -f3)
  echo $QUAY_IMAGE_TAG
  docker tag quay.io/pulp/pulp-operator:devel quay.io/pulp/pulp-operator:$QUAY_IMAGE_TAG
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

  make bundle-build
  make bundle-push

  make catalog-build
  make catalog-push
  docker images
fi

docker images
