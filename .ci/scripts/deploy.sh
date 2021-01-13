#!/bin/bash -e
#!/usr/bin/env bash

echo "Deploy pulp-operator"
sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

echo "Build and test pulp image"
cd $GITHUB_WORKSPACE/containers/
cp $GITHUB_WORKSPACE/.ci/ansible/vars.yaml vars/vars.yaml
pip install ansible
ansible-playbook -v build.yaml
cd $GITHUB_WORKSPACE
sudo -E ./up.sh
.ci/scripts/pulp-operator-check-and-wait.sh
.ci/scripts/pulp_file-tests.sh

echo "Deploy pulp latest"
sudo -E QUAY_REPO_NAME=pulp $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

echo "Deploy pulpcore latest"
sudo -E QUAY_REPO_NAME=pulpcore $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
