#!/bin/bash

APP_NAME="content-sources"  # name of app-sre "application" folder this component lives in
COMPONENT_NAME="content-sources-pulp-operator"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IMAGE="quay.io/cloudservices/pulp-operator"

# be explicit about what to build
DOCKERFILE=Dockerfile

IQE_PLUGINS="content-sources"
IQE_MARKER_EXPRESSION="api"
IQE_FILTER_EXPRESSION=""
IQE_ENV="ephemeral"
IQE_CJI_TIMEOUT="30m"

echo "WORKING DIR:"
pwd
# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/cicd-tools/master
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

echo "WORKING DIR:"
pwd
ls -la
make sdkbin OPERATOR_SDK_VERSION=v1.29.0 LOCALBIN=/tmp
ls -la /tmp/operator-sdk