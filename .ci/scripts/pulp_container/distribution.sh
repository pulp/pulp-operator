#!/usr/bin/env bash

DIST_NAME=$(head /dev/urandom | tr -dc a-z | head -c5)
DIST_BASE_PATH='test'

# Distributions are created asynchronously.
echo "Creating distribution \
  (name=$DIST_NAME, base_path=$DIST_BASE_PATH repository=$REPO_HREF)."
TASK_HREF=$(http POST $BASE_ADDR/pulp/api/v3/distributions/container/container/ \
  name=$DIST_NAME \
  base_path=$DIST_BASE_PATH \
  repository=$REPO_HREF | jq -r '.task')

# Poll the task (here we use a function defined in docs/_scripts/base.sh)
wait_until_task_finished $BASE_ADDR$TASK_HREF

echo "Setting DISTRIBUTION_HREF from the completed task."
# DISTRIBUTION_HREF is the pulp-api HREF, not the content app href
DISTRIBUTION_HREF=$(http $BASE_ADDR$TASK_HREF | jq -r '.created_resources | first')

echo "Inspecting Distribution."
http $BASE_ADDR$DISTRIBUTION_HREF
