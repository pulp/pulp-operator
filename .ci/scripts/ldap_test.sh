#!/bin/bash

set -eu

DEPLOYMENT_NAME="example-pulp-api"

echo "Verifying ldap authentication ..."
TEST_1=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ualice:alice localhost:24817/pulp/api/v3/content/)
TEST_2=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ualice:aaaaa localhost:24817/pulp/api/v3/content/)
TEST_3=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ubob:bob localhost:24817/pulp/api/v3/content/)
TEST_4=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ubob:aaaaa localhost:24817/pulp/api/v3/content/)
TEST_5=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ueve:eve localhost:24817/pulp/api/v3/content/)
TEST_6=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -so /dev/null -w "%{http_code}" -ueve:aaaaa localhost:24817/pulp/api/v3/content/)

declare -A tests
tests=( ["TEST_1"]="200" ["TEST_2"]="401" ["TEST_3"]="200" ["TEST_4"]="401" ["TEST_5"]="200" ["TEST_6"]="401" )


for test in ${!tests[@]} ; do
  echo -n "$test: ${tests[$test]} "
  if [[ ${!test} != ${tests[$test]} ]] ; then
    echo "[ERR]"
    exit 1
  else
    echo "[OK]"
  fi
done


echo "LDAP auth ok"
