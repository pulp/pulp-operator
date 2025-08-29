#!/bin/bash
set -euo pipefail

echo "Set context"
kubectl config set-context --current --namespace=pulp-operator-system

BACKUP_RESOURCE=repo-manager.pulpproject.org_v1_pulpbackup.yaml
RESTORE_RESOURCE=repo-manager.pulpproject.org_v1_pulprestore.yaml
PULP_CR=example-pulp

if [[ "$CI_TEST" == "true" ]]; then
  CUSTOM_RESOURCE=simple.yaml
elif [[ "$CI_TEST_STORAGE" == "azure" ]]; then
  CUSTOM_RESOURCE=simple.azure.ci.yaml
elif [[ "$CI_TEST_STORAGE" == "s3" ]]; then
  CUSTOM_RESOURCE=simple.s3.ci.yaml
fi

echo ::group::PRE_BACKUP_LOGS
journalctl --unit=pulp-operator -n 10000 --no-pager --output=cat
kubectl logs -l app.kubernetes.io/name=pulp-operator -c manager --tail=10000
echo ::endgroup::

kubectl apply -f config/samples/$BACKUP_RESOURCE
time kubectl wait --for condition=BackupComplete --timeout=800s -f config/samples/$BACKUP_RESOURCE

echo ::group::AFTER_BACKUP_LOGS
journalctl --unit=pulp-operator -n 10000 --no-pager --output=cat
kubectl logs -l app.kubernetes.io/name=pulp-operator -c manager --tail=10000
echo ::endgroup::

# kubectl delete --cascade=foreground -f config/samples/$CUSTOM_RESOURCE
kubectl delete -f config/samples/$CUSTOM_RESOURCE

# deleting resources that have no operator owerReference to better validate that
# restore controller will recreate them instead of "reusing" the older ones
kubectl delete secrets --all
kubectl delete pvc -l pulp_cr=$PULP_CR
kubectl wait --for=delete --timeout=300s -f config/samples/$CUSTOM_RESOURCE

kubectl apply -f config/samples/$RESTORE_RESOURCE
time kubectl wait --for condition=RestoreComplete --timeout=800s -f config/samples/$RESTORE_RESOURCE

echo ::group::AFTER_RESTORE_LOGS
journalctl --unit=pulp-operator -n 10000 --no-pager --output=cat
kubectl logs -l app.kubernetes.io/name=pulp-operator -c manager --tail=10000
echo ::endgroup::

sudo pkill -f "port-forward" || true
time kubectl wait --for condition=Pulp-Operator-Finished-Execution pulp.repo-manager.pulpproject.org/$PULP_CR --timeout=800s
kubectl get pods -o wide

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"
  if [[ "$CI_TEST" == "true" ]]; then
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

if [[ "$COMPONENT_TYPE" == "ingress" ]]; then
    SERVER=ingress.local
    export BASE_ADDR="http://$SERVER"
else
    export BASE_ADDR="http://$SERVER:$WEB_PORT"
fi
echo $BASE_ADDR

if [ -z "$(pip freeze | grep pulp-cli)" ]; then
  echo "Installing pulp-cli"
  pip install pulp-cli[pygments]
fi

API_ROOT=${API_ROOT:-"/pulp/"}

echo "Configuring pulp-cli"
mkdir -p ~/.config/pulp
cat > ~/.config/pulp/cli.toml << EOF
[cli]
base_url = "$BASE_ADDR"
api_root = "$API_ROOT"
verify_ssl = false
format = "json"
username = "admin"
password = "password"
EOF

cat ~/.config/pulp/cli.toml | tee ~/.config/pulp/settings.toml

pulp content list
CONTENT_LENGTH=$(pulp content list | jq length)
if [[ "$CONTENT_LENGTH" == "0" ]]; then
  echo "Empty content list"
  exit 1
fi
