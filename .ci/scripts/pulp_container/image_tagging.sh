#!/usr/bin/env bash

TAG_NAME='custom_tag'
MANIFEST_DIGEST=$(http $BASE_ADDR'/pulp/api/v3/content/container/manifests/?repository_version='$REPOVERSION_HREF \
  | jq -r '.results | first | .digest')

echo "Tagging the manifest."
TASK_URL=$(http POST $BASE_ADDR$REPO_HREF'tag/' tag=$TAG_NAME digest=$MANIFEST_DIGEST \
  | jq -r '.task')

wait_until_task_finished $BASE_ADDR$TASK_URL

echo "Getting a reference to a newly created tag."
CREATED_TAG=$(http $BASE_ADDR$TASK_URL \
  | jq -r '.created_resources | .[] | select(test("content"))')

echo "Display properties of the created tag."
http $BASE_ADDR$CREATED_TAG
