#!/usr/bin/env bash

# create a repository with the same name as the exported one
http ${BASE_ADDR}/pulp/api/v3/repositories/container/container/ \
  name="test/fixture" | jq -r ".pulp_href"

# import the exported repository stored in '/tmp/exports/test-fixture'
IMPORTER_HREF=$(http ${BASE_ADDR}/pulp/api/v3/importers/core/pulp/ \
  name="test/fixture" | jq -r ".pulp_href")
EXPORTED_REPO_PATH=$(find "/tmp/exports/test-fixture" -type f -name \
  "*.tar.gz" | head -n 1)
GROUP_HREF=$(http ${BASE_ADDR}${IMPORTER_HREF}imports/ \
  path=${EXPORTED_REPO_PATH} | jq -r ".task_group")
echo ${BASE_ADDR}${GROUP_HREF}
