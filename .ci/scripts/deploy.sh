#!/bin/bash -e
#!/usr/bin/env bash

QUAY_BOT_USERNAME=${QUAY_BOT_USERNAME:-pulp+github}

echo "$QUAY_BOT_PASSWORD" | docker login -u "$QUAY_BOT_USERNAME" --password-stdin quay.io
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
