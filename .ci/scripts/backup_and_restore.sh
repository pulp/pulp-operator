#!/bin/bash
set -euo pipefail

if command -v kubectl > /dev/null; then
  KUBECTL=$(command -v kubectl)
elif [ -x /usr/local/bin/kubectl ]; then
  KUBECTL=/usr/local/bin/kubectl
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

echo "Set context"
$KUBECTL config set-context --current --namespace=pulp-operator-system

if [[ "$CI_TEST" == "true" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.ci.yaml
  BACKUP_RESOURCE=pulpproject_v1beta1_pulpbackup_cr.ci.yaml
  RESTORE_RESOURCE=pulpproject_v1beta1_pulprestore_cr.ci.yaml
elif [[ "$CI_TEST" == "galaxy" ]]; then
  CUSTOM_RESOURCE=pulpproject_v1beta1_pulp_cr.galaxy.ci.yaml
  BACKUP_RESOURCE=pulpproject_v1beta1_pulpbackup_cr.ci.yaml
  RESTORE_RESOURCE=pulpproject_v1beta1_pulprestore_cr.ci.yaml
fi

echo ::group::PRE_BACKUP_LOGS
$KUBECTL logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

$KUBECTL apply -f config/samples/$BACKUP_RESOURCE
time $KUBECTL wait --for condition=BackupComplete --timeout=-1s -f config/samples/$BACKUP_RESOURCE

echo ::group::AFTER_BACKUP_LOGS
$KUBECTL logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

$KUBECTL delete --cascade=foreground -f config/samples/$CUSTOM_RESOURCE
$KUBECTL wait --for=delete -f config/samples/$CUSTOM_RESOURCE

$KUBECTL apply -f config/samples/$RESTORE_RESOURCE
time $KUBECTL wait --for condition=RestoreComplete --timeout=-1s -f config/samples/$RESTORE_RESOURCE

echo ::group::AFTER_RESTORE_LOGS
$KUBECTL logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

sudo pkill -f "port-forward" || true
time $KUBECTL wait --for condition=Pulp-Operator-Finished-Execution pulp/example-pulp --timeout=-1s

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"
  if [[ "$CI_TEST" == "true" ]] || [[ "$CI_TEST" == "galaxy" ]]; then
    services=$($KUBECTL get services)
    WEB_PORT=$( echo "$services" | awk -F '[ :/]+' '/web-svc/{print $5}')
    SVC_NAME=$( echo "$services" | awk -F '[ :/]+' '/web-svc/{print $1}')
    sudo pkill -f "port-forward" || true
    echo "port-forwarding service/$SVC_NAME $WEB_PORT:$WEB_PORT"
    $KUBECTL port-forward service/$SVC_NAME $WEB_PORT:$WEB_PORT &
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

if [[ "$CI_TEST" == "galaxy" ]]; then
  API_ROOT="/api/galaxy/pulp/"
fi
API_ROOT=${API_ROOT:-"/pulp/"}

if [ ! -f ~/.config/pulp/settings.toml ]; then
  echo "Configuring pulp-cli"
  mkdir -p ~/.config/pulp
  cat > ~/.config/pulp/cli.toml << EOF
[cli]
base_url = "$BASE_ADDR"
api_root = "$API_ROOT"
verify_ssl = false
format = "json"
EOF
fi

cat ~/.config/pulp/cli.toml | tee ~/.config/pulp/settings.toml

pulp content list
CONTENT_LENGTH=$(pulp content list | jq length)
if [[ "$CONTENT_LENGTH" == "0" ]]; then
  echo "Empty content list"
  exit 1
fi
