#!/bin/bash

set -eu

DEPLOYMENT_NAME="example-pulp-api"


echo "Validating otel-collector-config file ..."
kubectl exec -c otel-collector-sidecar deployment/$DEPLOYMENT_NAME -- /otelcol validate --config=/etc/otelcol-contrib/otel-collector-config.yaml

echo "Verifying if metrics service is available ..."
HTTP_STATUS=$(kubectl exec deployment/$DEPLOYMENT_NAME -- curl -sw "%{http_code}" -o /dev/null localhost:8889/metrics)
if [[ "$HTTP_STATUS" != 200 ]]; then
  echo "error: ${HTTP_STATUS}"
  exit 1
fi

echo "Verifying if \"http_server\" string is found in metrics endpoint ..."
kubectl exec deployment/$DEPLOYMENT_NAME -- curl -s localhost:8889/metrics | grep http_server  &>/dev/null

echo "Telemetry ok"
