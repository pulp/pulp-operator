#!/usr/bin/env bash

TAG_NAME='custom_tag'

echo "Untagging a manifest which is labeled with ${TAG_NAME}"
TASK_URL=$(http POST $BASE_ADDR$REPO_HREF'untag/' tag=$TAG_NAME \
  | jq -r '.task')

wait_until_task_finished $BASE_ADDR$TASK_URL

echo "Getting a reference to all removed tags."
REPO_VERSION=$(http $BASE_ADDR$TASK_URL \
  | jq -r '.created_resources | first')
REMOVED_TAGS=$(http $BASE_ADDR$REPO_VERSION \
  | jq -r '.content_summary | .removed | ."container.tag" | .href')

echo "List removed tags from the latest repository version."
http $BASE_ADDR$REMOVED_TAGS
