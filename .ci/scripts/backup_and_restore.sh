#!/bin/bash
set -euo pipefail

echo "Set context"
kubectl config set-context --current --namespace=pulp-operator-system

BACKUP_RESOURCE=backup_cr.ci.yaml
RESTORE_RESOURCE=restore_cr.ci.yaml

if [[ "$CI_TEST" == "true" ]]; then
  CUSTOM_RESOURCE=simple.yaml
elif [[ "$CI_TEST" == "galaxy" && "$CI_TEST_STORAGE" == "filesystem" ]]; then
  CUSTOM_RESOURCE=galaxy.yaml
elif [[ "$CI_TEST" == "galaxy" && "$CI_TEST_STORAGE" == "azure" ]]; then
  CUSTOM_RESOURCE=galaxy.azure.ci.yaml
elif [[ "$CI_TEST" == "galaxy" && "$CI_TEST_STORAGE" == "s3" ]]; then
  CUSTOM_RESOURCE=galaxy.s3.ci.yaml
fi

echo ::group::PRE_BACKUP_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

kubectl apply -f config/samples/$BACKUP_RESOURCE
time kubectl wait --for condition=BackupComplete --timeout=-1s -f config/samples/$BACKUP_RESOURCE

echo ::group::AFTER_BACKUP_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

kubectl delete --cascade=foreground -f config/samples/$CUSTOM_RESOURCE
kubectl wait --for=delete -f config/samples/$CUSTOM_RESOURCE

kubectl apply -f config/samples/$RESTORE_RESOURCE
time kubectl wait --for condition=RestoreComplete --timeout=-1s -f config/samples/$RESTORE_RESOURCE

echo ::group::AFTER_RESTORE_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
echo ::endgroup::

sudo pkill -f "port-forward" || true
time kubectl wait --for condition=Pulp-Operator-Finished-Execution pulp/example-pulp --timeout=-1s

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"
  if [[ "$CI_TEST" == "true" ]] || [[ "$CI_TEST" == "galaxy" ]]; then
    services=$(kubectl get services)
    WEB_PORT=$( echo "$services" | awk -F '[ :/]+' '/web-svc/{print $5}')
    SVC_NAME=$( echo "$services" | awk -F '[ :/]+' '/web-svc/{print $1}')
    sudo pkill -f "port-forward" || true
    echo "port-forwarding service/$SVC_NAME $WEB_PORT:$WEB_PORT"
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
