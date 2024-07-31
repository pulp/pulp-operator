#!/usr/bin/env bash

podman pull ghcr.io/pulp/test-fixture-1:manifest_a

# push a tagged image to the registry
podman login ${REGISTRY_ADDR} -u admin -p password --tls-verify=false
podman tag ghcr.io/pulp/test-fixture-1:manifest_a \
  ${REGISTRY_ADDR}/test/fixture:manifest_a
podman push ${REGISTRY_ADDR}/test/fixture:manifest_a --tls-verify=false

# a repository of the push type is automatically created
REPOSITORY_HREF=$(pulp container repository -t push show \
  --name "test/fixture" | jq -r ".pulp_href")

# export the repository to the directory '/tmp/exports/test-fixture'
EXPORTER_HREF=$(http ${BASE_ADDR}/pulp/api/v3/exporters/core/pulp/ \
  name=both repositories:="[\"${REPOSITORY_HREF}\"]" \
  path=/tmp/exports/test-fixture | jq -r ".pulp_href")
TASK_HREF=$(http POST ${BASE_ADDR}${EXPORTER_HREF}exports/ | jq -r ".task")
wait_until_task_finished ${BASE_ADDR}${TASK_HREF}
