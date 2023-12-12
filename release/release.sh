#!/bin/bash

set -eu

cd $PULP_OPERATOR_SOURCE_PATH

function new_release {
  echo "Creating a new branch ..."
  git stash save --include-untracked "stashing due to release $PULP_OPERATOR_RELEASE_VERSION process ..."
  git checkout -B $GIT_RELEASE_BRANCH
  git fetch $GIT_UPSTREAM_REMOTE_NAME
  git rebase ${GIT_UPSTREAM_REMOTE_NAME}/main
  
  echo "Running towncrier ..."
  towncrier --yes --version $PULP_OPERATOR_RELEASE_VERSION
  
  echo "Ensuring the manifests are updated ..."
  make generate manifests bundle
  
  echo "Commiting changes ..."
  git commit -am "Release version $PULP_OPERATOR_RELEASE_VERSION" -m "[noissue]"
}


function bump_version {
  echo "Bumping main.go version ..."
  sed -i -E "s/(pulp-operator version:) .*/\1 ${PULP_OPERATOR_DEV_VERSION}\")/g" main.go

  echo "Bumping Makefile operator's version ..."
  sed -i -E "s/^(VERSION \?=) .*/\1 ${PULP_OPERATOR_DEV_VERSION}/g" Makefile

  echo "Bumping the tag used in bundle-upgrade pipeline  ..."
  sed -i -E "s/(ref:) ${PULP_OPERATOR_REPLACE_VERSION}/\1 ${PULP_OPERATOR_RELEASE_VERSION}/g" .github/workflows/ci.yml

  echo "Updating the manifests with the changes ..."
  make generate manifests bundle

  echo "Commiting changes ..."
  git commit -am "Bump version from Makefile to $PULP_OPERATOR_DEV_VERSION" -m "[noissue]"
}

function stash_files {
  pushd $1
  git stash save --include-untracked "stashing due to release $PULP_OPERATOR_RELEASE_VERSION process ..."
  popd
}


function operatorhub {

  CATALOG_PATH="${CATALOG_PATH:-$OPERATORHUB_REPO_PATH}"
  CATALOG_DIR=${CATALOG_PATH}/operators/pulp-operator/${PULP_OPERATOR_RELEASE_VERSION}
  stash_files $CATALOG_PATH
  mkdir -p ${CATALOG_DIR}
  RELEASE_COMMIT=$(git --no-pager log -1 --pretty=format:"%h" --grep="Release version $PULP_OPERATOR_RELEASE_VERSION")

  git checkout $RELEASE_COMMIT

  pushd $CATALOG_PATH
  BRANCH="${BRANCH:-$GIT_OPERATORHUB_RELEASE_BRANCH}"
  UPSTREAM="${UPSTREAM:-$GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME}"

  git checkout -B $BRANCH
  git fetch $UPSTREAM
  git rebase ${UPSTREAM}/main

  cp -a ${PULP_OPERATOR_SOURCE_PATH}/bundle/* ${CATALOG_DIR}/
  CSV_FILE=${CATALOG_DIR}/manifests/pulp-operator.clusterserviceversion.yaml
  sed -i "s#containerImage: quay.io/pulp/pulp-operator:devel#containerImage: quay.io/pulp/pulp-operator:${PULP_OPERATOR_RELEASE_VERSION}#g" $CSV_FILE
  echo "  replaces: pulp-operator.${PULP_OPERATOR_REPLACE_VERSION}" >> $CSV_FILE

  echo "Commiting changes ..."
  git add operators/pulp-operator/
  git commit -sm "operator pulp-operator ($PULP_OPERATOR_RELEASE_VERSION)"
  popd

  git checkout $GIT_RELEASE_BRANCH
}

function redhat_catalog {
    CATALOG_PATH=$REDHAT_CATALOG_REPO_PATH UPSTREAM=$GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME operatorhub
}

new_release
bump_version
operatorhub
redhat_catalog

echo -e "[\e[32mOK\e[0m] All tasks finished!"