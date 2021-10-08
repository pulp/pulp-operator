#!/bin/bash -e
#!/usr/bin/env bash

KUBE="minikube"
if [[ "$1" == "--kind" ]] || [[ "$1" == "-k" ]]; then
  KUBE="kind"
  echo "Running $KUBE"
fi

sudo -E kubectl get pods -o wide
sudo -E kubectl get deployments

if [[ "$KUBE" == "minikube" ]]; then
  echo ::group::METRICS
  sudo -E kubectl top pods
  sudo -E kubectl describe node minikube
  echo ::endgroup::
fi

echo ::group::OPERATOR_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-operator -c manager --tail=10000
echo ::endgroup::

echo ::group::PULP_API_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-api --tail=10000
echo ::endgroup::

echo ::group::PULP_CONTENT_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-content --tail=10000
echo ::endgroup::

echo ::group::PULP_WORKER_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-worker --tail=10000
echo ::endgroup::

echo ::group::PULP_RESOURCE_MANAGER_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-resource-manager --tail=10000
echo ::endgroup::

echo ::group::PULP_WEB_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=nginx --tail=10000
echo ::endgroup::
