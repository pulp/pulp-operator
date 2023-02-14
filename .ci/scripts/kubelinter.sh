#!/usr/bin/env bash
# coding=utf-8

set -euo pipefail

RELEASE_INFO=$(curl --silent --show-error --fail https://api.github.com/repos/stackrox/kube-linter/releases/latest)
RELEASE_NAME=$(echo "${RELEASE_INFO}" | jq --raw-output ".name")
LOCATION=$(echo "${RELEASE_INFO}" \
  | jq --raw-output ".assets[].browser_download_url" \
  | grep --fixed-strings kube-linter-linux.tar.gz)
TARGET=kube-linter-linux-${RELEASE_NAME}.tar.gz
# Skip downloading release if downloaded already, e.g. when the action is used multiple times.
if [ ! -e $TARGET ]; then
  curl --silent --show-error --fail --location --output $TARGET "$LOCATION"
  tar -xf $TARGET
fi
mkdir -p lint
kubectl get ${TEST:-pulp,pulpbackup,pulprestore},pvc,configmap,serviceaccount,secret,networkpolicy,ingress,service,deployment,statefulset,hpa,job,cronjob -o yaml > ./lint/k8s-all.yaml
./kube-linter lint ./lint --config .ci/assets/kubernetes/.kube-linter.yaml
