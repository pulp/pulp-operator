#!/usr/bin/env bash

# deploy/cluster_role_binding.yaml specify the namespace: default
# The default namespace on molecule is: osdk-test
# For running the molecule test we should:
# - Option 1: change the namespace at deploy/cluster_role_binding.yaml to: osdk-test
# - Option 2: change the namespace at molecule test to: default
# As molecule provides TEST_OPERATOR_NAMESPACE we will use the 2nd option:
export TEST_OPERATOR_NAMESPACE=default

echo "Starting molecule test"
molecule test -s test-local
