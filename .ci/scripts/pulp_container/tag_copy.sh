#!/usr/bin/env bash

echo "Create a task to copy a tag to the repo."
TASK_HREF=$(http POST $BASE_ADDR$SECOND_REPO_HREF'copy_tags/' \
  source_repository=$REPO_HREF \
  names:="[\"manifest_a\"]" \
  | jq -r '.task')

# Poll the task (here we use a function defined in docs/_scripts/base.sh)
wait_until_task_finished $BASE_ADDR$TASK_HREF

# After the task is complete, it gives us a new repository version
TAG_COPY_VERSION=$(http $BASE_ADDR$TASK_HREF | jq -r '.created_resources | first')

echo "Inspect RepositoryVersion."
http $BASE_ADDR$TAG_COPY_VERSION
