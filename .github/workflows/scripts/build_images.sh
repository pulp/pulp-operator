#!/bin/bash -e
#!/usr/bin/env bash

echo "Build pulp/pulpcore images"
cd $GITHUB_WORKSPACE/containers/
if [[ "$CI_TEST" == "galaxy" ]]; then
  cp $GITHUB_WORKSPACE/.ci/ansible/galaxy/vars.yaml vars/vars.yaml
else
  cp $GITHUB_WORKSPACE/.ci/ansible/pulp/vars.yaml vars/vars.yaml
fi
sed -i "s/podman/docker/g" common_tasks.yaml
pip install ansible

if [[ -z "${QUAY_EXPIRE+x}" ]]; then
  ansible-playbook -v build.yaml
else
  sed -i "s/latest/${QUAY_IMAGE_TAG}/g" vars/vars.yaml
  echo "Building tag: ${QUAY_IMAGE_TAG}"
  ansible-playbook -v build.yaml --extra-vars "quay_expire=${QUAY_EXPIRE}"
fi
cd $GITHUB_WORKSPACE

echo "Build web images"
cd $GITHUB_WORKSPACE/containers/
if [[ "$CI_TEST" == "galaxy" ]]; then
  cp $GITHUB_WORKSPACE/.ci/ansible/galaxy/web/vars.yaml vars/vars.yaml
else
  cp $GITHUB_WORKSPACE/.ci/ansible/pulp/web/vars.yaml vars/vars.yaml
fi
sed -i "s/podman/docker/g" common_tasks.yaml

if [[ -z "${QUAY_EXPIRE+x}" ]]; then
  ansible-playbook -v build.yaml
else
  sed -i "s/latest/${QUAY_IMAGE_TAG}/g" vars/vars.yaml
  echo "Building tag: ${QUAY_IMAGE_TAG}"
  ansible-playbook -v build.yaml --extra-vars "quay_expire=${QUAY_EXPIRE}"
fi
cd $GITHUB_WORKSPACE

docker images
