#!/usr/bin/env bash
REPO_NAME=$(head /dev/urandom | tr -dc a-z | head -c5)

echo "Creating a new repository named $REPO_NAME."
REPO_HREF=$(http POST $BASE_ADDR/pulp/api/v3/repositories/container/container/ name=$REPO_NAME \
  | jq -r '.pulp_href')

echo "Inspecting repository."
http $BASE_ADDR$REPO_HREF
