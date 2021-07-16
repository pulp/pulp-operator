#!/usr/bin/env bash
# coding=utf-8

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"
  if [[ "$CI_TEST" == "true" ]]; then
    SVC_NAME="example-pulp-web-svc"
    WEB_PORT="24880"
    kubectl port-forward service/$SVC_NAME $WEB_PORT:$WEB_PORT &
  fi
fi

# From the pulp-server/pulp-api config-map
echo "machine $SERVER
login admin
password password\
" > ~/.netrc

export BASE_ADDR="http://$SERVER:$WEB_PORT"
echo $BASE_ADDR

if [ -z "$(pip freeze | grep pulp-cli)" ]; then
  echo "Installing pulp-cli"
  pip install pulp-cli[pygments]
fi

if [ ! -f ~/.config/pulp/settings.toml ]; then
  echo "Configuring pulp-cli"
  mkdir -p ~/.config/pulp
  cat > ~/.config/pulp/cli.toml << EOF
[cli]
base_url = "$BASE_ADDR"
verify_ssl = false
format = "json"
EOF
fi

cat ~/.config/pulp/cli.toml | tee ~/.config/pulp/settings.toml

pushd pulp_file/docs/_scripts
# Let's only do sync tests.
# So as to check that Pulp can work in containers, including writing to disk.
# If the upload tests are simpler in the long run, just use them.
#
# If the master branch tests fail, run the stable tests.
# The git command is to checkout the newest stag, which should be the
# stable release.
# Temporary workaround until we replace with pulp-smash.
timeout 5m bash -x docs_check_sync_publish.sh || {
  echo "Master branch of pulp_file tests failed. Using newest tag (stable release.)"
  git checkout $(git describe --tags `git rev-list --tags --max-count=1`)
  timeout 5m bash -x docs_check_sync_publish.sh
}

