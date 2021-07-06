#!/bin/bash

KUBE="k3s"
SERVER=$(hostname)
WEB_PORT="24817"
if [[ "$1" == "--minikube" ]] || [[ "$1" == "-m" ]]; then
  KUBE="minikube"
  SERVER="localhost"
  if [[ "$CI_TEST" == "true" ]]; then
    SVC_NAME="example-pulp-web-svc"
    WEB_PORT="24880"
    kubectl port-forward service/$SVC_NAME $WEB_PORT:$WEB_PORT &
  fi
fi

BASE_ADDR="http://$SERVER:$WEB_PORT"
echo $BASE_ADDR
echo "Base Address: $BASE_ADDR"
REPOS=( "published" "staging" "rejected" "community" "rh-certified" )
REPO_RESULTS=()

echo "Waiting ..."
sleep 10

TOKEN=$(curl --location --request POST "$BASE_ADDR/api/galaxy/v3/auth/token/" --header 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' --silent | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")
echo $TOKEN

echo "Testing ..."

for repo in "${REPOS[@]}"
do
	# echo $repo
    COLLECTION_URL="$BASE_ADDR/api/galaxy/content/$repo/v3/collections/"
    # echo $COLLECTION_URL
    HTTP_CODE=$(curl --write-out "%{http_code}\n" -H "Authorization:Token $TOKEN" $COLLECTION_URL --silent --output /dev/null)
    # echo $HTTP_CODE
    REPO_RESULTS+=($HTTP_CODE)
done

GALAXY_INIT_RESULT=0
ITER=0
for code in "${REPO_RESULTS[@]}"
do
    echo "${REPOS[$ITER]} $code"
    ITER=$((ITER + 1))
    if [[ $code != 200 ]]; then
        GALAXY_INIT_RESULT=$ITER
    fi
done

exit $GALAXY_INIT_RESULT
