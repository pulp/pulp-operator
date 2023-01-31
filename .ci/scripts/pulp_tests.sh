#!/usr/bin/env bash
# coding=utf-8
set -euo pipefail

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "${1-}" == "--minikube" ]] || [[ "${1-}" == "-m" ]]; then
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

# Install pulp-cli the same way we do in pulp_ansible repo (to avoid tests with version mismatch, for example)
# https://github.com/pulp/pulp_ansible/blob/5778d86ae51738578f7c5f00214b5ccb8aa1ee45/.github/workflows/scripts/before_install.sh#L96-L102
git clone --depth=1 https://github.com/pulp/pulp-cli.git
pushd pulp-cli
pip install .
popd

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

pulp status | jq

pushd pulp_ansible/docs/_scripts
timeout 5m bash -x quickstart.sh || {
  YLATEST=$(git ls-remote --heads https://github.com/pulp/pulp_ansible.git | grep -o "[[:digit:]]\.[[:digit:]]*" | sort -V | tail -1)
  git fetch --depth=1 origin heads/$YLATEST:$YLATEST
  git checkout $YLATEST
  timeout 5m bash -x quickstart.sh
}
popd

pushd pulp_container/docs/_scripts
timeout 5m bash -x docs_check.sh || {
  YLATEST=$(git ls-remote --heads https://github.com/pulp/pulp_container.git | grep -o "[[:digit:]]\.[[:digit:]]*" | sort -V | tail -1)
  git fetch --depth=1 origin heads/$YLATEST:$YLATEST
  git checkout $YLATEST
  timeout 5m bash -x docs_check.sh
}
popd
