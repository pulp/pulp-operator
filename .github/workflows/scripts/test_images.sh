#!/bin/bash -e
#!/usr/bin/env bash

KUBE_FLAG=""
if
  [ "$1" = "--minikube" ] || [ "$1" = "-m" ]; then
  KUBE_FLAG="-m"
fi

cd $GITHUB_WORKSPACE

echo "Test pulp/pulpcore images"
if [[ -n "${QUAY_EXPIRE}" ]]; then
  echo "LABEL quay.expires-after=${QUAY_EXPIRE}d" >> ./build/Dockerfile
fi
sudo -E ./up.sh
.ci/scripts/pulp-operator-check-and-wait.sh $KUBE_FLAG
if [[ "$CI_TEST" == "galaxy" ]]; then
  CI_TEST=true .ci/scripts/galaxy_ng-tests.sh -m
else
  .ci/scripts/retry.sh 3 ".ci/scripts/pulp_tests.sh -m"
fi

docker images
