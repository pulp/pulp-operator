#!/usr/bin/env bash
DEST_REPO_NAME=$(head /dev/urandom | tr -dc a-z | head -c5)

echo "Create a second repository so we can add content to it."
SECOND_REPO_HREF=$(http POST $BASE_ADDR/pulp/api/v3/repositories/container/container/ name=$DEST_REPO_NAME \
  | jq -r '.pulp_href')

echo "Inspect repository."
http $BASE_ADDR$SECOND_REPO_HREF
