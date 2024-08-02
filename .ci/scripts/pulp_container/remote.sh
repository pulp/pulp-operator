#!/usr/bin/env bash
REMOTE_NAME=$(head /dev/urandom | tr -dc a-z | head -c5)

echo "Creating $REMOTE_NAME remote that points to an external source of container images."
REMOTE_HREF=$(http POST $BASE_ADDR/pulp/api/v3/remotes/container/container/ \
    name=$REMOTE_NAME \
    url='https://registry-1.docker.io' \
    upstream_name='pulp/test-fixture-1' | jq -r '.pulp_href')

echo "Inspecting new Remote."
http $BASE_ADDR$REMOTE_HREF
