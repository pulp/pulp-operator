#!/usr/bin/env bash

echo "Create a task to copy all manifests from source to destination repo."
TASK_HREF=$(http POST $BASE_ADDR$SECOND_REPO_HREF'copy_manifests/' \
  source_repository=$REPO_HREF \
  | jq -r '.task')

# Poll the task (here we use a function defined in docs/_scripts/base.sh)
wait_until_task_finished $BASE_ADDR$TASK_HREF

# After the task is complete, it gives us a new repository version
MANIFEST_COPY_VERSION=$(http $BASE_ADDR$TASK_HREF | jq -r '.created_resources | first')

echo "Inspect RepositoryVersion."
http $BASE_ADDR$MANIFEST_COPY_VERSION
