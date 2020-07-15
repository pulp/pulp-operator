#!/usr/bin/env bash

# Avoid nodePort error:
# nodePort: Invalid value: 24816 The range of valid ports is 30000-32767
# nodePort: Invalid value: 24817 The range of valid ports is 30000-32767
find ./roles/* -exec sed -i 's/24816/31816/g' {} \;
find ./roles/* -exec sed -i 's/24817/31817/g' {} \;

# deploy/cluster_role_binding.yaml specify the namespace: default
# The default namespace on molecule is: osdk-test
# For running the molecule test we should:
# - Option 1: change the namespace at deploy/cluster_role_binding.yaml to: osdk-test
# - Option 2: change the namespace at molecule test to: default
# As molecule provides TEST_OPERATOR_NAMESPACE we will use the 2nd option:
export TEST_OPERATOR_NAMESPACE=default

echo "Starting molecule test"
molecule test -s test-local
