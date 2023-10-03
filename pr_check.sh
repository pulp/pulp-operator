#!/bin/bash

# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/cicd-tools/master
curl -s $CICD_URL/bootstrap.sh > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

make sdkbin OPERATOR_SDK_VERSION=v1.29.0 LOCALBIN=/tmp
