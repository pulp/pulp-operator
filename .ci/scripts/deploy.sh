#!/bin/bash -e
#!/usr/bin/env bash

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
