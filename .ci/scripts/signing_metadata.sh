#!/bin/bash

set -xe

# get pulp admin password
PULP_ADM_PWD=$(kubectl get secret/example-pulp-admin-password -ojsonpath='{.data.password}'|base64 -d)

# verify the list of signing services (keeping it in a different variable to make troubleshooting/debug easier)
SIGNING_SVC=$(kubectl exec deployment/example-pulp-api -- curl  -u admin:$PULP_ADM_PWD -sL localhost:24817/pulp/api/v3/signing-services/)

# get only the count of services found
SVC_COUNT=$(echo $SIGNING_SVC | jq .count)

# check if the 2 services were found
if [[ $SVC_COUNT != 2 ]] ; then
  echo "Could not find all signing services!"
  exit 1
fi

# check if the the gpg key is in the api's keyring
kubectl exec deployment/example-pulp-api -- gpg -k joe@foo.bar 2>/dev/null

# check if the the gpg key is in the worker's keyring
kubectl exec deployment/example-pulp-worker -- gpg -k joe@foo.bar 2>/dev/null

