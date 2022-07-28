#!/usr/bin/env bash

set -ex #fail in case of non zero return

CI_TEST=${CI_TEST:-pulp}
API_ROOT=${API_ROOT:-"/pulp/"}
OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE:-"pulp-operator-system"}
PULP_INSTANCE="ocp-example"
INGRESS_DEFAULT_DOMAIN=$(oc get ingresses.config/cluster -o jsonpath={.spec.domain})
ROUTE_HOST=${1:-"${PULP_INSTANCE}.${INGRESS_DEFAULT_DOMAIN}"}

OUTPUT_TEMPLATE='{{.spec.host}} {{.spec.path}} {{.spec.port.targetPort}} {{.spec.tls.termination}} {{.spec.to.name}}'
# check root path
root_path=( $(oc -n $OPERATOR_NAMESPACE get route $PULP_INSTANCE -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${root_path[0]} != "$ROUTE_HOST" ] ; then exit 1 ; fi
if [ ${root_path[1]} != "/" ] ; then exit 2 ; fi
if [ ${root_path[2]} != "api-24817" ] ; then exit 3 ; fi
if [ ${root_path[3]} != "edge" ] ; then exit 4 ; fi
if [ ${root_path[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 5 ; fi
echo "[OK] / path ..."

# check /api/v3/ path
api_v3_path=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-api-v3 -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${api_v3_path[0]} != "$ROUTE_HOST" ] ; then exit 6 ; fi
if [ ${api_v3_path[1]} != "/pulp/api/v3/" ] ; then exit 7 ; fi
if [ ${api_v3_path[2]} != "api-24817" ] ; then exit 8 ; fi
if [ ${api_v3_path[3]} != "edge" ] ; then exit 9 ; fi
if [ ${api_v3_path[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 10 ; fi
echo "[OK] /api/v3/ path ..."

# check /auth/login
auth_login=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-auth -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${auth_login[0]} != "$ROUTE_HOST" ] ; then exit 11 ; fi
if [ ${auth_login[1]} != "/auth/login/" ] ; then exit 12 ; fi
if [ ${auth_login[2]} != "api-24817" ] ; then exit 13 ; fi
if [ ${auth_login[3]} != "edge" ] ; then exit 14 ; fi
if [ ${auth_login[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 15 ; fi
echo "[OK] /auth/login/ path ..."

# check /pulp/content/
core_content=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-content -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${core_content[0]} != "$ROUTE_HOST" ] ; then exit 16 ; fi
if [ ${core_content[1]} != "/pulp/content/" ] ; then exit 17 ; fi
if [ ${core_content[2]} != "content-24816" ] ; then exit 18 ; fi
if [ ${core_content[3]} != "edge" ] ; then exit 19 ; fi
if [ ${core_content[4]} != "${PULP_INSTANCE}-content-svc" ] ; then exit 20 ; fi
echo "[OK] /pulp/content/ path ..."

# check /pulp_ansible/galaxy/
ansible_galaxy=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-ansible-pulp-ansible-galaxy -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${ansible_galaxy[0]} != "$ROUTE_HOST" ] ; then exit 21 ; fi
if [ ${ansible_galaxy[1]} != "/pulp_ansible/galaxy/" ] ; then exit 22 ; fi
if [ ${ansible_galaxy[2]} != "api-24817" ] ; then exit 23 ; fi
if [ ${ansible_galaxy[3]} != "edge" ] ; then exit 24 ; fi
if [ ${ansible_galaxy[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 25 ; fi
echo "[OK] /pulp/galaxy/ path ..."

# check /extensions/v2/
extensions_v2=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-container-extensions-v2 -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${extensions_v2[0]} != "$ROUTE_HOST" ] ; then exit 26 ; fi
if [ ${extensions_v2[1]} != "/extensions/v2/" ] ; then exit 27 ; fi
if [ ${extensions_v2[2]} != "api-24817" ] ; then exit 28 ; fi
if [ ${extensions_v2[3]} != "edge" ] ; then exit 29 ; fi
if [ ${extensions_v2[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 30 ; fi
echo "[OK] /extensions/v2/ path ..."

# check /pulp/container/
container=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-container-pulp-container -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${container[0]} != "$ROUTE_HOST" ] ; then exit 31 ; fi
if [ ${container[1]} != "/pulp/container/" ] ; then exit 32 ; fi
if [ ${container[2]} != "content-24816" ] ; then exit 33 ; fi
if [ ${container[3]} != "edge" ] ; then exit 34 ; fi
if [ ${container[4]} != "${PULP_INSTANCE}-content-svc" ] ; then exit 35 ; fi
echo "[OK] /pulp/container/ path ..."

# check /token/
token=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-container-token -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${token[0]} != "$ROUTE_HOST" ] ; then exit 36 ; fi
if [ ${token[1]} != "/token/" ] ; then exit 37 ; fi
if [ ${token[2]} != "api-24817" ] ; then exit 38 ; fi
if [ ${token[3]} != "edge" ] ; then exit 39 ; fi
if [ ${token[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 40 ; fi
echo "[OK] /token/ path ..."

# check /v2/
v2=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-container-v2 -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${v2[0]} != "$ROUTE_HOST" ] ; then exit 41 ; fi
if [ ${v2[1]} != "/v2/" ] ; then exit 42 ; fi
if [ ${v2[2]} != "api-24817" ] ; then exit 43 ; fi
if [ ${v2[3]} != "edge" ] ; then exit 44 ; fi
if [ ${v2[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 45 ; fi
echo "[OK] /v2/ path ..."

# check /pulp_cookbook/content/
cookbook=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-cookbook-pulp-cookbook-content -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${cookbook[0]} != "$ROUTE_HOST" ] ; then exit 46 ; fi
if [ ${cookbook[1]} != "/pulp_cookbook/content/" ] ; then exit 47 ; fi
if [ ${cookbook[2]} != "content-24816" ] ; then exit 48 ; fi
if [ ${cookbook[3]} != "edge" ] ; then exit 49 ; fi
if [ ${cookbook[4]} != "${PULP_INSTANCE}-content-svc" ] ; then exit 50 ; fi
echo "[OK] /pulp_cookbook/content/ path ..."

# check /pulp_npm/content/
npm=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-npm-pulp-npm-content -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${npm[0]} != "$ROUTE_HOST" ] ; then exit 51 ; fi
if [ ${npm[1]} != "/pulp_npm/content/" ] ; then exit 52 ; fi
if [ ${npm[2]} != "content-24816" ] ; then exit 53 ; fi
if [ ${npm[3]} != "edge" ] ; then exit 54 ; fi
if [ ${npm[4]} != "${PULP_INSTANCE}-content-svc" ] ; then exit 55 ; fi
echo "[OK] /pulp_npm/content/ path ..."

# check /pypi/
pypi=( $(oc -n $OPERATOR_NAMESPACE get route ${PULP_INSTANCE}-python-pypi -ogo-template="$OUTPUT_TEMPLATE") )
if [ ${pypi[0]} != "$ROUTE_HOST" ] ; then exit 56 ; fi
if [ ${pypi[1]} != "/pypi/" ] ; then exit 57 ; fi
if [ ${pypi[2]} != "api-24817" ] ; then exit 58 ; fi
if [ ${pypi[3]} != "edge" ] ; then exit 59 ; fi
if [ ${pypi[4]} != "${PULP_INSTANCE}-api-svc" ] ; then exit 60 ; fi
echo "[OK] /pypi/ path ..."
