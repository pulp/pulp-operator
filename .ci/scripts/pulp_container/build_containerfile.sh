#!/usr/bin/env bash

echo "Create a task that will build a container image from a Containerfile."

TASK_HREF=$(http --form POST :$REPO_HREF'build_image/' containerfile@./Containerfile \
artifacts="{\"$ARTIFACT_HREF\": \"foo/bar/example.txt\"}"  | jq -r '.task')

# Poll the task (here we use a function defined in docs/_scripts/base.sh)
wait_until_task_finished $BASE_ADDR$TASK_HREF

# After the task is complete, it gives us a new repository version
echo "Set REPOVERSION_HREF from finished task."
REPOVERSION_HREF=$(http $BASE_ADDR$TASK_HREF| jq -r '.created_resources | first')

echo "Inspecting RepositoryVersion."
http $BASE_ADDR$REPOVERSION_HREF
