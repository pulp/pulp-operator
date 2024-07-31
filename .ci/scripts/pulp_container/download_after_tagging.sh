#!/usr/bin/env bash

TAG_NAME='custom_tag'

DIST_NAME=$(head /dev/urandom | tr -dc a-z | head -c5)
DIST_BASE_PATH='tag'

echo "Publishing the latest repository."
TASK_URL=$(http POST $BASE_ADDR/pulp/api/v3/distributions/container/container/ \
  name=$DIST_NAME base_path=$DIST_BASE_PATH repository=$REPO_HREF \
  | jq -r '.task')

wait_until_task_finished $BASE_ADDR$TASK_URL

DISTRIBUTION_HREF=$(http $BASE_ADDR$TASK_URL \
  | jq -r '.created_resources | first')
REGISTRY_PATH=$(http $BASE_ADDR$DISTRIBUTION_HREF \
  | jq -r '.registry_path')

echo "Running ${REGISTRY_PATH}:${TAG_NAME}."
sudo docker login -u admin -p password $REGISTRY_PATH
sudo docker run $REGISTRY_PATH:$TAG_NAME
