#!/bin/bash -e
#!/usr/bin/env bash

docker images

KUBE="minikube"
if [[ "$1" == "--kind" ]] || [[ "$1" == "-k" ]]; then
  KUBE="kind"
  echo "Running $KUBE"
fi

kubectl get pods -o wide
kubectl get pods -o go-template='{{range .items}} {{.metadata.name}} {{range .status.containerStatuses}} {{.lastState.terminated.exitCode}} {{end}}{{"\n"}} {{end}}'
kubectl get deployments

if [[ "$KUBE" == "minikube" ]]; then
  echo ::group::METRICS
  kubectl top pods || true
  kubectl describe node minikube || true
  echo ::endgroup::
  echo ::group::MINIKUBE_LOGS
  minikube logs -n 10000
  echo ::endgroup::
fi

echo ::group::EVENTS
kubectl get events --sort-by='.metadata.creationTimestamp'
echo ::endgroup::

echo ::group::OBJECTS
kubectl get pulp,pvc,configmap,serviceaccount,secret,networkpolicy,ingress,service,deployment,statefulset,hpa,job,cronjob -o yaml
echo ::endgroup::

echo ::group::OPERATOR_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-operator -c manager --tail=10000
echo ::endgroup::

echo ::group::PULP_API_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-api --tail=10000
echo ::endgroup::

echo ::group::PULP_CONTENT_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-content --tail=10000
echo ::endgroup::

echo ::group::PULP_WORKER_LOGS
kubectl logs -l app.kubernetes.io/name=pulp-worker --tail=10000
echo ::endgroup::

echo ::group::PULP_WEB_LOGS
kubectl logs -l app.kubernetes.io/name=nginx --tail=10000
echo ::endgroup::

echo ::group::POSTGRES
kubectl logs -l app.kubernetes.io/name=postgres --tail=10000
echo ::endgroup::

echo "Status endpoint"
http --follow --timeout 30 --check-status --pretty format --print hb http://localhost:24880/pulp/api/v3/status/ || true
