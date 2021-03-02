#!/bin/bash -e
#!/usr/bin/env bash

KIND_URL=https://kind.sigs.k8s.io/dl/v0.8.1/kind-linux-amd64

if [[ "$OSTYPE" == "darwin"* ]]; then
  KIND_URL=https://kind.sigs.k8s.io/dl/v0.8.1/kind-darwin-amd64
fi

curl -Lo ./kind $KIND_URL
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind


# deploy/cluster_role_binding.yaml specify the namespace: default
# The default namespace on molecule is: osdk-test
# For running the molecule test we should:
# - Option 1: change the namespace at deploy/cluster_role_binding.yaml to: osdk-test
# - Option 2: change the namespace at molecule test to: default
# As molecule provides TEST_OPERATOR_NAMESPACE we will use the 2nd option:
if [[ "$OSTYPE" == "darwin"* ]]; then
  sed -i "" "s/default/osdk-test/g" deploy/cluster_role_binding.yaml
else
  sed -i "s/default/osdk-test/g" deploy/cluster_role_binding.yaml
fi

echo "Starting molecule test"
molecule -v test -s kind
