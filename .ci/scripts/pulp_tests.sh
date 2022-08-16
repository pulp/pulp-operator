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

if [ -z "$(pip3 freeze | grep pulp-cli)" ]; then
  echo "Installing pulp-cli"
  pip3 install --user pulp-cli[pygments]
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
echo 127.0.0.1   example-pulp-web-svc.pulp-operator-go-system.svc.cluster.local | sudo tee -a /etc/hosts

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
