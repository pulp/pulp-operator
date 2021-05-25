#!/bin/bash -e
#!/usr/bin/env bash

KUBE_FLAG=""
if
  [ "$1" = "--minikube" ] || [ "$1" = "-m" ]; then
  KUBE_FLAG="-m"
fi


echo "Build pulp/pulpcore images"
cd $GITHUB_WORKSPACE/containers/
if [[ "$CI_TEST" == "galaxy" ]]; then
  cp $GITHUB_WORKSPACE/.ci/ansible/galaxy/vars.yaml vars/vars.yaml
else
  cp $GITHUB_WORKSPACE/.ci/ansible/vars.yaml vars/vars.yaml
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
cp $GITHUB_WORKSPACE/.ci/ansible/web/vars.yaml vars/vars.yaml
sed -i "s/podman/docker/g" common_tasks.yaml

if [[ -z "${QUAY_EXPIRE+x}" ]]; then
  ansible-playbook -v build.yaml
else
  sed -i "s/latest/${QUAY_IMAGE_TAG}/g" vars/vars.yaml
  echo "Building tag: ${QUAY_IMAGE_TAG}"
  ansible-playbook -v build.yaml --extra-vars "quay_expire=${QUAY_EXPIRE}"
fi
cd $GITHUB_WORKSPACE

echo "Test pulp/pulpcore images"
if [[ -n "${QUAY_EXPIRE}" ]]; then
  echo "LABEL quay.expires-after=${QUAY_EXPIRE}d" >> ./build/Dockerfile
fi
sudo -E ./up.sh
.ci/scripts/pulp-operator-check-and-wait.sh $KUBE_FLAG
if [[ "$CI_TEST" == "galaxy" ]]; then
  .ci/scripts/galaxy_ng-tests.sh -m
else
  .ci/scripts/retry.sh 3 ".ci/scripts/pulp_file-tests.sh -m"
fi

docker images

if [[ "$CI_TEST" == "galaxy" ]]; then
  echo "Deploy galaxy latest"
  sudo -E QUAY_REPO_NAME=galaxy $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
  echo "Deploy galaxy-web latest"
  sudo -E QUAY_REPO_NAME=galaxy-web $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
else
  echo "Deploy pulp latest"
  sudo -E QUAY_REPO_NAME=pulp $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

  echo "Deploy pulpcore latest"
  sudo -E QUAY_REPO_NAME=pulpcore $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

  echo "Deploy pulp-web latest"
  sudo -E QUAY_REPO_NAME=pulp-web $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
fi

if [[ -z "${QUAY_EXPIRE+x}" ]]; then
  echo "Deploy pulp-operator"
  eval $(minikube -p minikube docker-env)
  sudo -E $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh
fi

export QUAY_IMAGE_TAG=$(python -c 'import yaml; print(yaml.safe_load(open("deploy/olm-catalog/pulp-operator/manifests/pulp-operator.clusterserviceversion.yaml"))["spec"]["version"])')
docker build -f bundle.Dockerfile -t quay.io/pulp/pulp-operator-bundle:${QUAY_IMAGE_TAG} .
sudo -E QUAY_REPO_NAME=pulp-operator-bundle $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh


wget https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/latest-4.7/opm-linux.tar.gz
tar xvf opm-linux.tar.gz
sudo mv opm /usr/local/bin/opm
sudo chmod +x /usr/local/bin/opm


opm index add -c docker --bundles quay.io/pulp/pulp-operator-bundle:${QUAY_IMAGE_TAG} --tag quay.io/pulp/pulp-index:${QUAY_IMAGE_TAG}
sudo -E QUAY_REPO_NAME=pulp-index $GITHUB_WORKSPACE/.ci/scripts/quay-push.sh

docker images
