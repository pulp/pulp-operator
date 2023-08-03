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

echo ::group::EVENTS
kubectl get events --sort-by='.metadata.creationTimestamp'
echo ::endgroup::

echo ::group::OBJECTS
kubectl get ${TEST:-pulp},pvc,configmap,serviceaccount,secret,networkpolicy,ingress,service,deployment,statefulset,hpa,job,cronjob -o yaml -l owner!=helm
echo ::endgroup::

echo ::group::OPERATOR_LOGS
journalctl --unit=pulp-operator -n 10000 --no-pager --output=cat
kubectl logs -l app.kubernetes.io/component=operator -c manager --tail=10000
echo ::endgroup::

echo ::group::DESCRIBE_JOBS
kubectl describe jobs
echo ::endgroup::

echo ::group::MIGRATION_JOB_LOGS
for i in $(kubectl get jobs -oname -l app.kubernetes.io/component=migration) ; do echo "=== $i ===" ; kubectl logs --timestamps $i ; done
echo ::endgroup::

echo ::group::ADMIN_PWD_JOB_LOGS
for i in $(kubectl get jobs -oname -l app.kubernetes.io/component=reset-admin-password) ; do echo "=== $i ===" ; kubectl logs --timestamps  $i ; done
echo ::endgroup::

echo ::group::INITCONTAINER_API_LOGS
kubectl logs --timestamps -cinit-container -l app.kubernetes.io/component=api --tail=10000
echo ::endgroup::

echo ::group::INITCONTAINER_CONTENT_LOGS
kubectl logs --timestamps -cinit-container -l app.kubernetes.io/component=content --tail=10000
echo ::endgroup::

echo ::group::INITCONTAINER_WORKER_LOGS
kubectl logs --timestamps -cinit-container -l app.kubernetes.io/component=worker --tail=10000
echo ::endgroup::

echo ::group::PULP_API_PODS
kubectl describe pods -l app.kubernetes.io/component=api
echo ::endgroup::

echo ::group::PULP_API_LOGS
kubectl logs --timestamps -l app.kubernetes.io/component=api --tail=10000
echo ::endgroup::

echo ::group::PULP_CONTENT_PODS
kubectl describe pods -l app.kubernetes.io/component=content
echo ::endgroup::

echo ::group::PULP_CONTENT_LOGS
kubectl logs --timestamps -l app.kubernetes.io/component=content --tail=10000
echo ::endgroup::

echo ::group::PULP_WORKER_PODS
kubectl describe pods -l app.kubernetes.io/component=worker
echo ::endgroup::

echo ::group::PULP_WORKER_LOGS
kubectl logs --timestamps -l app.kubernetes.io/component=worker --tail=10000
echo ::endgroup::

echo ::group::PULP_WEB_PODS
kubectl describe pods -l app.kubernetes.io/component=webserver
echo ::endgroup::

echo ::group::PULP_WEB_LOGS
kubectl logs -l app.kubernetes.io/component=webserver --tail=10000
echo ::endgroup::

echo ::group::POSTGRES
kubectl logs -l app.kubernetes.io/component=database --tail=10000
echo ::endgroup::

if [[ "$KUBE" == "minikube" ]]; then
  echo ::group::METRICS
  kubectl top pods || true
  kubectl describe node minikube || true
  echo ::endgroup::
  echo ::group::MINIKUBE_LOGS
  minikube logs -n 10000
  echo ::endgroup::
fi

if [[ "$INGRESS_TYPE" == "ingress" ]]; then
    export BASE_ADDR="http://ingress.local"
else
    export BASE_ADDR="http://nodeport.local:30000"
fi

echo "Status endpoint"
http --follow --timeout 30 --check-status --pretty format --print hb $BASE_ADDR/pulp/api/v3/status/ || true
