#!/bin/bash
#!/usr/bin/env bash

echo "Deploy pulp-operator"
sudo $TRAVIS_BUILD_DIR/.travis/quay-push.sh

echo "Build pulp and pulpcore"
cd $TRAVIS_BUILD_DIR/../pulpcore/containers/
cp $TRAVIS_BUILD_DIR/.travis/vars.yaml vars/vars.yaml
pip install ansible
ansible-playbook -v build.yaml

echo "Deploy pulp latest"
sudo QUAY_REPO_NAME=pulp $TRAVIS_BUILD_DIR/.travis/quay-push.sh

echo "Deploy pulpcore latest"
sudo QUAY_REPO_NAME=pulpcore $TRAVIS_BUILD_DIR/.travis/quay-push.sh
