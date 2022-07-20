#!/usr/bin/env bash

set -e #fail in case of non zero return

CI_TEST=${CI_TEST:-pulp}
API_ROOT=${API_ROOT:-"/pulp/"}

sed -i 's/kubectl/oc/g' Makefile
make deploy
oc apply -f .ci/assets/kubernetes/pulp-admin-password.secret.yaml

if [[ "$CI_TEST" == "galaxy" ]]; then
  oc apply -f config/samples/pulpproject_v1beta1_pulp_cr.galaxy.ocp.ci.yaml
else
  oc apply -f config/samples/pulpproject_v1beta1_pulp_cr.ocp.ci.yaml
fi

oc wait --for condition=Pulp-Operator-Finished-Execution pulp/ocp-example --timeout=-1s
API_POD=$(oc get pods -l app.kubernetes.io/component=api -oname)
oc exec ${API_POD} -- curl -L http://localhost:24817${API_ROOT}api/v3/status/
