#!/usr/bin/env bash
# coding=utf-8

KUBE="k3s"
SERVER=$(hostname)
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"

fi

# From the pulp-server/pulp-api config-map
echo "machine $SERVER
login admin
password password\
" > ~/.netrc

export BASE_ADDR="http://$SERVER:24817"

pushd pulp_file/docs/_scripts
# Let's only do sync tests.
# So as to check that Pulp can work in containers, including writing to disk.
# If the upload tests are simpler in the long run, just use them.
#
# If the master branch tests fail, run the stable tests.
# The git command is to checkout the newest stag, which should be the
# stable release.
# Temporary workaround until we replace with pulp-smash.
timeout 5m bash -x docs_check_sync_publish.sh || {
  echo "Master branch of pulp_file tests failed. Using newest tag (stable release.)"
  git checkout $(git describe --tags `git rev-list --tags --max-count=1`)
  timeout 5m bash -x docs_check_sync_publish.sh
}

