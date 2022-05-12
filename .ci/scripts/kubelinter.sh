#!/usr/bin/env bash
# coding=utf-8

set -euo pipefail

RELEASE_INFO=$(curl --silent --show-error --fail https://api.github.com/repos/stackrox/kube-linter/releases/latest)
RELEASE_NAME=$(echo "${RELEASE_INFO}" | jq --raw-output ".name")
LOCATION=$(echo "${RELEASE_INFO}" \
  | jq --raw-output ".assets[].browser_download_url" \
  | grep --fixed-strings kube-linter-linux)
LINTER=kube-linter-linux-${RELEASE_NAME}
# Skip downloading release if downloaded already, e.g. when the action is used multiple times.
if [ ! -e $LINTER ]; then
  curl --silent --show-error --fail --location --output $LINTER "$LOCATION"
  chmod +x $LINTER
fi
mkdir -p lint
sudo -E kubectl get pvc,configmap,serviceaccount,secret,networkpolicy,ingress,service,deployment,statefulset,hpa,job,cronjob -o yaml > ./lint/k8s-all.yaml
./$LINTER lint ./lint --config .ci/assets/kubernetes/.kube-linter.yaml
