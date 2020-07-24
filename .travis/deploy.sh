#!/bin/bash
#!/usr/bin/env bash

echo "Deploy pulp-operator"
sudo $TRAVIS_BUILD_DIR/.travis/quay-push.sh

echo "Build and test pulp image"
cd $TRAVIS_BUILD_DIR/containers/
cp $TRAVIS_BUILD_DIR/.travis/vars.yaml vars/vars.yaml
pip install ansible
ansible-playbook -v build.yaml
cd $TRAVIS_BUILD_DIR
sudo ./up.sh
.travis/pulp-operator-check-and-wait.sh
.travis/pulp_file-tests.sh

echo "Deploy pulp latest"
sudo QUAY_REPO_NAME=pulp $TRAVIS_BUILD_DIR/.travis/quay-push.sh

echo "Deploy pulpcore latest"
sudo QUAY_REPO_NAME=pulpcore $TRAVIS_BUILD_DIR/.travis/quay-push.sh
