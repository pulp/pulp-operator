#!/bin/bash -e
#!/usr/bin/env bash

KUBE_FLAG=""
if
  [ "$1" = "--minikube" ] || [ "$1" = "-m" ]; then
  KUBE_FLAG="-m"
fi

if command -v kubectl > /dev/null; then
  KUBECTL=$(command -v kubectl)
elif [ -x /usr/local/bin/kubectl ]; then
  KUBECTL=/usr/local/bin/kubectl
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

cd $GITHUB_WORKSPACE

echo "Test pulp/pulpcore images"
if [[ -n "${QUAY_EXPIRE}" ]]; then
  echo "LABEL quay.expires-after=${QUAY_EXPIRE}d" >> ./build/Dockerfile
fi
sudo -E ./up.sh
time $KUBECTL wait --for condition=Pulp-Operator-Finished-Execution pulp/example-pulp --timeout=-1s
if [[ "$CI_TEST" == "galaxy" ]]; then
  CI_TEST=true .ci/scripts/galaxy_ng-tests.sh -m
else
  .ci/scripts/retry.sh 3 ".ci/scripts/pulp_tests.sh -m"
fi

docker images
