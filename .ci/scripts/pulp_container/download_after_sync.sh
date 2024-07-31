#!/usr/bin/env bash

CONTAINER_TAG='manifest_a'

echo "Setting REGISTRY_PATH, which can be used directly with the Docker Client."
REGISTRY_PATH=$(http $BASE_ADDR$DISTRIBUTION_HREF | jq -r '.registry_path')

echo "Next we pull and run the image from pulp"
echo "$REGISTRY_PATH:$CONTAINER_TAG"
sudo docker login -u admin -p password $REGISTRY_PATH
sudo docker run $REGISTRY_PATH:$CONTAINER_TAG
