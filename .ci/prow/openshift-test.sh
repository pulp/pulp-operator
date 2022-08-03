#!/usr/bin/env bash

set -euo pipefail

CI_TEST=${CI_TEST:-pulp}
API_ROOT=${API_ROOT:-"/pulp/"}

show_logs() {
  oc get pods -o wide
  oc get routes -o wide
  echo "======================== Operator ========================"
  oc logs -l app.kubernetes.io/name=pulp-operator -c pulp-manager --tail=10000
  echo "======================== API ========================"
  oc logs -l app.kubernetes.io/name=pulp-api --tail=10000
  echo "======================== Content ========================"
  oc logs -l app.kubernetes.io/name=pulp-content --tail=10000
  echo "======================== Worker ========================"
  oc logs -l app.kubernetes.io/name=pulp-worker --tail=10000
  echo "======================== Postgres ========================"
  oc logs -l app.kubernetes.io/name=postgres --tail=10000
  echo "======================== Events ========================"
  oc get events --sort-by='.metadata.creationTimestamp'
  exit 1
}

oc apply -f .ci/assets/kubernetes/pulp-admin-password.secret.yaml

ROUTE_HOST="pulpci.$(oc get ingresses.config/cluster -o jsonpath={.spec.domain})"
echo $ROUTE_HOST

if [[ "$CI_TEST" == "galaxy" ]]; then
  CR_FILE=config/samples/pulpproject_v1beta1_pulp_cr.galaxy.ocp.ci.yaml
else
  CR_FILE=config/samples/pulpproject_v1beta1_pulp_cr.ocp.ci.yaml
fi

sed -i "s/route_host_placeholder/$ROUTE_HOST/g" $CR_FILE
oc apply -f $CR_FILE
# oc wait --for condition=Pulp-Routes-Ready --timeout=-1s -f $CR_FILE || show_logs

# while [[ $(oc get pods -l app.kubernetes.io/component=api -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do
#   echo "STATUS: Still waiting on pods to transition to running state."
#   oc get pods -o wide
#   sleep 1
# done
# sleep 1
# oc get pods -o wide
# echo "Check status endpoint:"
# API_POD=$(oc get pods -l app.kubernetes.io/component=api -oname)
# oc exec ${API_POD} -- curl -L http://localhost:24817${API_ROOT}api/v3/status/ || show_logs

# BASE_ADDR="https://${ROUTE_HOST}"
# echo ${BASE_ADDR}${API_ROOT}api/v3/status/
# curl --insecure --fail --location ${BASE_ADDR}${API_ROOT}api/v3/status/ || show_logs
