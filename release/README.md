# ABOUT

The release container is used to make the process of releasing a new version of `pulp-operator` easier. <br/>
The idea of using a container, instead of just running the release.sh script, is to avoid issues because of different installations and to make it easier to run it (no need to install several apps). For example, running `make bundle` with different versions of `operator-sdk` will generate different bundle manifests, or towncrier docs can have incompatibility with newer versions.

# RUNNING

## PRE-REQS
* if you don't have a local copy of the repos yet, fork and clone them:
```sh
GITHUB_SPACE="git-hyagi"
git clone --depth 1 git@github.com:${GITHUB_SPACE}/pulp-operator /tmp/pulp-operator
git clone --depth 1 git@github.com:${GITHUB_SPACE}/community-operators/ /tmp/community-operators
git clone --depth 1 git@github.com:${GITHUB_SPACE}/community-operators-prod/ /tmp/community-operators-prod/
```

* configure the repositories upstreams:
```sh
REMOTE_NAME="upstream"
cd /tmp/pulp-operator
git remote add $REMOTE_NAME https://github.com/pulp/pulp-operator.git

cd /tmp/community-operators
git remote add $REMOTE_NAME https://github.com/k8s-operatorhub/community-operators.git

cd /tmp/community-operators-prod
git remote add $REMOTE_NAME https://github.com/redhat-openshift-ecosystem/community-operators-prod.git
```

## BUILD

* build the container image (run this command from inside the `release` directory of pulp-operator source):
```sh
podman build -t pulp-operator-release .
```

## RUN
```sh
GIT_RELEASE_BRANCH=release-beta-4
PULP_OPERATOR_REPLACE_VERSION=1.0.0-beta.3
PULP_OPERATOR_RELEASE_VERSION=1.0.0-beta.4
PULP_OPERATOR_DEV_VERSION=1.0.0-beta.5
GITHUB_SSH_KEY=~/.ssh/github
PULP_OPERATOR_REPO_PATH=/tmp/pulp-operator/
OPERATORHUB_REPO_PATH=/tmp/community-operators/
REDHAT_CATALOG_REPO_PATH=/tmp/community-operators-prod/
GIT_UPSTREAM_REMOTE_NAME=upstream
GIT_REMOTE_NAME=origin
GIT_OPERATORHUB_RELEASE_BRANCH="pulp-operator-$PULP_OPERATOR_RELEASE_VERSION"
GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME=upstream
GIT_OPERATORHUB_REMOTE_NAME=origin
GIT_REDHAT_CATALOG_RELEASE_BRANCH="pulp-operator-$PULP_OPERATOR_RELEASE_VERSION"
GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME=upstream
GIT_REDHAT_CATALOG_REMOTE_NAME=origin

podman run -dit --name pulp-operator-release \
  -v ~/.gitconfig:/root/.gitconfig:Z \
  -v ${GITHUB_SSH_KEY}:/root/.ssh/id_rsa:Z \
  -v ${PULP_OPERATOR_REPO_PATH}:/app/pulp-operator/:Z \
  -v ${OPERATORHUB_REPO_PATH}:/app/community-operators/:Z \
  -v ${REDHAT_CATALOG_REPO_PATH}/:/app/community-operators-prod/:Z \
  -e GIT_RELEASE_BRANCH \
  -e GIT_REMOTE_BRANCH \
  -e PULP_OPERATOR_REPLACE_VERSION \
  -e PULP_OPERATOR_RELEASE_VERSION \
  -e PULP_OPERATOR_DEV_VERSION \
  -e GIT_OPERATORHUB_RELEASE_BRANCH \
  -e GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME \
  -e GIT_OPERATORHUB_REMOTE_NAME \
  -e GIT_REDHAT_CATALOG_RELEASE_BRANCH \
  -e GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME \
  -e GIT_REDHAT_CATALOG_REMOTE_NAME \
  pulp-operator-release
```


|ENV|DESCRIPTION|
|---|-----------|
|GITHUB_SSH_KEY|ssh private key to authenticate to github. For example: ~/.ssh/id_rsa|
|PULP_OPERATOR_REPO_PATH|host path of `pulp-operator` source repository. For example: ~/pulp/pulp-operator|
|OPERATORHUB_REPO_PATH|host path of `operatorhub catalog` source repository. For example: ~/pulp/community-operators|
|REDHAT_CATALOG_REPO_PATH|host path of `redhat catalog` source repository. For example: ~/pulp/community-operators-prod|
|GIT_RELEASE_BRANCH|name of the branch that will be created on `pulp-operator` repo for this new release. For example: release-beta-4|
|GIT_UPSTREAM_REMOTE_NAME|`pulp-operator` upstream git remote name. For example: upstream|
|GIT_REMOTE_NAME|`pulp-operator` fork git remote name. For example: origin|
|PULP_OPERATOR_REPLACE_VERSION|`pulp-operator` version to be replaced by this release. For example:1.0.0-beta.3|
|PULP_OPERATOR_RELEASE_VERSION|`pulp-operator` release version. For example: 1.0.0-beta.4|
|PULP_OPERATOR_DEV_VERSION|`pulp-operator` development version (version that will be worked on after this release). For example: 1.0.0-beta.5|
|GIT_OPERATORHUB_RELEASE_BRANCH|name of the release branch that will be created on `operatorhub catalog`. For example: "pulp-operator-$PULP_OPERATOR_RELEASE_VERSION"|
|GIT_OPERATORHUB_UPSTREAM_REMOTE_NAME|`operatorhub catalog` upstream git remote name. For example: upstream|
|GIT_OPERATORHUB_REMOTE_NAME|`operatorhub catalog` fork git remote name. For example: origin|
|GIT_REDHAT_CATALOG_RELEASE_BRANCH|name of the release branch that will be created on `redhat catalog`. For example: "pulp-operator-$PULP_OPERATOR_RELEASE_VERSION"|
|GIT_REDHAT_CATALOG_UPSTREAM_REMOTE_NAME|`redhat catalog` upstream git remote name. For example: upstream|
|GIT_REDHAT_CATALOG_REMOTE_NAME|`redhat catalog` fork git remote name. For example: origin|


# CHECKING

After the container finishes its execution, each repository should have a new branch with the changes.

### pulp-operator
The container should have created 2 new commits in the new branch:
```
$ git log --oneline upstream/main..
321e29f (HEAD -> release-beta-4) Bump version from Makefile to 1.0.0-beta.5
8a7c783 Release version 1.0.0-beta.4
```

The **release commit** should have the changes from towncrier (`CHANGES/*` files and `CHANGES.md`). <br/>
The **bump Makefile** commit should modify the following lines: `ref`, `VERSION`, `name`, `image`, `version`, `newTag`, and `pulp-operator version`.
For example:
```diff
$ git log -p -1 -U0
* 866022b (HEAD -> release-beta-4) [2023-12-11 12:33:11 +0000] git-hyagi  Bump version from Makefile to 1.0.0-beta.5|
  diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
  index bc884a4..2e5acc9 100644
  --- a/.github/workflows/ci.yml
  +++ b/.github/workflows/ci.yml
  @@ -90 +90 @@ jobs:
  -          ref: 1.0.0-beta.3
  +          ref: 1.0.0-beta.4
  diff --git a/Makefile b/Makefile
  index cc267ca..2fa2299 100644
  --- a/Makefile
  +++ b/Makefile
  @@ -13 +13 @@ WEB_IMAGE ?= quay.io/pulp/pulp-web
  -VERSION ?= 1.0.0-beta.4
  +VERSION ?= 1.0.0-beta.5
  diff --git a/bundle/manifests/pulp-operator.clusterserviceversion.yaml b/bundle/manifests/pulp-operator.clusterserviceversion.yaml
  index 22fd9d3..b55de46 100644
  --- a/bundle/manifests/pulp-operator.clusterserviceversion.yaml
  +++ b/bundle/manifests/pulp-operator.clusterserviceversion.yaml
  @@ -171 +171 @@ metadata:
  -    createdAt: "2023-12-11T12:33:08Z"
  +    createdAt: "2023-12-11T12:33:11Z"
  @@ -177 +177 @@ metadata:
  -  name: pulp-operator.v1.0.0-beta.4
  +  name: pulp-operator.v1.0.0-beta.5
  @@ -1304 +1304 @@ spec:
  -                image: quay.io/pulp/pulp-operator:v1.0.0-beta.4
  +                image: quay.io/pulp/pulp-operator:v1.0.0-beta.5
  @@ -1616 +1616 @@ spec:
  -  version: 1.0.0-beta.4
  +  version: 1.0.0-beta.5
  diff --git a/config/manager/kustomization.yaml b/config/manager/kustomization.yaml
  index 351751e..b734a9a 100644
  --- a/config/manager/kustomization.yaml
  +++ b/config/manager/kustomization.yaml
  @@ -16 +16 @@ images:
  -  newTag: v1.0.0-beta.4
  +  newTag: v1.0.0-beta.5
  diff --git a/main.go b/main.go
  index d5edeee..f025f3c 100644
  --- a/main.go
  +++ b/main.go
  @@ -180 +180 @@ func main() {
  -   setupLog.Info("pulp-operator version: 1.0.2-beta.4")
  + setupLog.Info("pulp-operator version: 1.0.0-beta.5")
```


### community-operators and community-operators-prod

The container should have created a new commit in a new branch with the following files:
```
$ git l -1 --name-only
* ff46725b3 (HEAD -> pulp-operator-<RELEASE_VERSION>) [2023-12-11 12:33:32 +0000] git-hyagi  operator pulp-operator (<RELEASE_VERSION>)|
  operators/pulp-operator/<RELEASE_VERSION>/manifests/pulp-operator-controller-manager-metrics-service_v1_service.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/pulp-operator-manager-config_v1_configmap.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/pulp-operator-manager-rolebinding_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/pulp-operator-metrics-reader_rbac.authorization.k8s.io_v1_clusterrole.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/pulp-operator.clusterserviceversion.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/repo-manager.pulpproject.org_pulpbackups.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/repo-manager.pulpproject.org_pulprestores.yaml
  operators/pulp-operator/<RELEASE_VERSION>/manifests/repo-manager.pulpproject.org_pulps.yaml
  operators/pulp-operator/<RELEASE_VERSION>/metadata/annotations.yaml
  operators/pulp-operator/<RELEASE_VERSION>/tests/scorecard/config.yaml
```

Since these are all new files, the git diff will not be that helpful, but make sure to double-check the CSV manifest (compare the current release with the one to be replaced):
```
$ diff -y --suppress-common-lines operators/pulp-operator/1.0.0-beta.{3,4}/manifests/pulp-operator.clusterserviceversion.yaml
    containerImage: quay.io/pulp/pulp-operator:v1.0.0-beta.3  |     containerImage: quay.io/pulp/pulp-operator:1.0.0-beta.4
    createdAt: "2023-12-04T14:48:41Z"                         |     createdAt: "2023-12-11T12:33:08Z"
  name: pulp-operator.v1.0.0-beta.3                           |   name: pulp-operator.v1.0.0-beta.4
                image: quay.io/pulp/pulp-operator:v1.0.0-beta |                 image: quay.io/pulp/pulp-operator:v1.0.0-beta
  version: 1.0.0-beta.3                                       |   version: 1.0.0-beta.4
  replaces: pulp-operator.v1.0.0-beta.2                       |   replaces: pulp-operator.1.0.0-beta.3
```
