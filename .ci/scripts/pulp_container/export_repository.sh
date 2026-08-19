#!/usr/bin/env bash

podman pull quay.io/pulp/test-fixture-1:manifest_a

# push a tagged image to the registry
podman login ${REGISTRY_ADDR} -u admin -p password --tls-verify=false
podman tag quay.io/pulp/test-fixture-1:manifest_a \
  ${REGISTRY_ADDR}/test/fixture:manifest_a
podman push ${REGISTRY_ADDR}/test/fixture:manifest_a --tls-verify=false

# a repository is automatically created (new versions use ContainerRepository,
# older versions use ContainerPushRepository)
REPOSITORY_HREF=$(pulp container repository show \
  --name "test/fixture" 2>/dev/null | jq -r ".pulp_href // empty")
if [ -z "$REPOSITORY_HREF" ]; then
  REPOSITORY_HREF=$(pulp container repository -t push show \
    --name "test/fixture" | jq -r ".pulp_href")
fi

# export the repository to the directory '/tmp/exports/test-fixture'
EXPORTER_HREF=$(http ${BASE_ADDR}/pulp/api/v3/exporters/core/pulp/ \
  name=both repositories:="[\"${REPOSITORY_HREF}\"]" \
  path=/tmp/exports/test-fixture | jq -r ".pulp_href")
TASK_HREF=$(http POST ${BASE_ADDR}${EXPORTER_HREF}exports/ | jq -r ".task")
wait_until_task_finished ${BASE_ADDR}${TASK_HREF}
