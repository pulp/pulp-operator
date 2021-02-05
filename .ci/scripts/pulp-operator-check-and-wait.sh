#!/usr/bin/env bash
# coding=utf-8

# pulp-operator-check-and-wait.sh:
# 1. Check that pulp-operator was successfully deployed on top of K8s
# 2. Wait for pulp-operator to be deployed to the point that pulp-api is able to
# serve requests.
#
# Currently only tested with k3s rather than a full K8s implementation.
# Uses generic K8s logic though.

storage_debug() {
  echo "VOLUMES:"
  sudo $KUBECTL get pvc
  sudo $KUBECTL get pv
  df -h
  sudo $KUBECTL -n local-path-storage get pod
  sudo $KUBECTL -n local-path-storage logs $STORAGE_POD
}

# CentOS 7 /etc/sudoers does not include /usr/local/bin
# Which k3s installs to.
# But we do not want to prevent other possible kubectl implementations.
# So use the user's current PATH to find and save the location of kubectl.

# CentOS 7 /etc/sudoers , and non-interactive shells (vagrant provisions)
# do not include /usr/local/bin , Which k3s installs to.
# But we do not want to break other possible kubectl implementations by
# hardcoding /usr/local/bin/kubectl .
# So if kubectl is in the user's PATH, preserve the filepath for sudo.
# And if kubectl is not in the PATH, assume /usr/local/bin/kubectl .
if command -v kubectl > /dev/null; then
  KUBECTL=$(command -v kubectl)
elif [ -x /usr/local/bin/kubectl ]; then
  KUBECTL=/usr/local/bin/kubectl
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

# Once the services are both up, the pods will be in a Pending state.
# Before the services are both up, the pods may not exist at all.
# So check for the services being up 1st.
for tries in {0..30}; do
  services=$(sudo $KUBECTL get services)
  if [[ $(echo "$services" | grep -c NodePort) -eq 2 ]]; then
    # parse string like this. 30805 is the external port
    # pulp-api     NodePort    10.43.170.79   <none>        24817:30805/TCP   0s
    API_PORT=$( echo "$services" | awk -F '[ :/]+' '/pulp-api/{print $6}')
    echo "SERVICES:"
    echo "$services"
    break
  else
    if [[ $tries -eq 30 ]]; then
      echo "ERROR 2: 1 or more external services never came up"
      echo "NAMESPACES:"
      sudo $KUBECTL get namespaces
      echo "SERVICES:"
      echo "$services"
      if [ -x "$(command -v docker)" ]; then
        echo "DOCKER IMAGE CACHE:"
        sudo docker images
      fi
      echo "PODS:"
      sudo $KUBECTL get pods -o wide
      storage_debug
      exit 2
    fi
  fi
  sleep 5
done

# This needs to be down here. Otherwise, the storage pod may not be
# up in time.
STORAGE_POD=$(sudo $KUBECTL -n local-path-storage get pod | awk '/local-path-provisioner/{print $1}')

# NOTE: Before the pods can be started, they must be downloaded/cached from
# quay.io .
# Therefore, this wait is highly dependent on network speed.
for tries in {0..180}; do
  pods=$(sudo $KUBECTL get pods -o wide)
  if [[ $(echo "$pods" | grep -c -v -E "STATUS|Running") -eq 0 ]]; then
    echo "PODS:"
    echo "$pods"
    API_NODE=$( echo "$pods" | awk -F '[ :/]+' '/pulp-api/{print $8}')
    break
  else
    # Often after 30 tries (150 secs), not all of the pods are running yet.
    # Let's keep Travis from ending the build by outputting.
    if [[ $(( tries % 30 )) == 0 ]]; then
      echo "STATUS: Still waiting on pods to transition to running state."
      echo "PODS:"
      echo "$pods"
      if [ -x "$(command -v docker)" ]; then
        echo "DOCKER IMAGE CACHE:"
        sudo docker images
      fi
    fi
    if [[ $tries -eq 180 ]]; then
      echo "ERROR 3: Pods never all transitioned to Running state"
      storage_debug
      exit 3
    fi
  fi
  sleep 5
done

# Later tests in other scripts will use localhost:24817, which was not a safe
# assumption at the time this script was originally written.
URL=http://$API_NODE:$API_PORT/pulp/api/v3/status/
echo "URL:"
echo $URL

if ! [ -x "$(command -v http)" -a -x "$(command -v jq)" ]; then
  echo 'WARNING 100: `http` & `jq` not installed'
  echo ""
  echo "Pulp may or may not be successfully running, but not immediately", and
  echo "this script can not perform its remaining checks."
  echo ""
  echo "Wait a few minutes (or longer if slow system/internet) and check manually:"
  echo "http://$API_NODE:$API_PORT/pulp/api/v3/status/"
  exit 100
fi

# Sometimes 30 tries is not enough for the service to actually come up
# Until it does:
# http: error: Request timed out (5.0s).
#
# --pretty format --print hb almost make it behave as if it were not redirected
for tries in {0..120}; do
  if [[ $tries -eq 120 ]]; then
    echo "ERROR 4: Status page never accessible or returning success"
    storage_debug
    exit 4
  fi
  output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
  rc=$?
  if echo "$output" | grep -e "Errno 111" -e "error(104" ; then
    # if connection refused, httpie does not wait 5 seconds
    sleep 5
  elif echo "$output" | grep "Request timed out" ; then
    continue
  elif [[ $rc ]] ; then
    echo "Successfully got the status page after _roughly_ $((tries * 5)) seconds"
    echo "$output"
    break
  fi
done

messages=(
    "pulp-api is connected to the database"
    "pulp-api is connected to redis"
    "Content app is online"
    "1 or more worker is online"
)

error_messages=(
    "ERROR 5: pulp-api never connected to the database"
    "ERROR 6: pulp-api never connected to redis"
    "ERROR 7: Content app never came online"
    "ERROR 8: Worker(s) never came online"
)

tests=(
    "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .database_connection.connected)" = "true"
    "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .redis_connection.connected)" = "true"
    "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .online_content_apps)" != "[]"
    "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .online_workers)" != "[]"
)

for iteration in {5..8};do
    index=$(($iteration - 5))
    for tries in {0..120}; do
    if [[ $tries -eq 120 ]]; then
        echo ${error_messages[$index]}
        storage_debug
        echo "$output"
        exit $iteration
    fi
    output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
    if [[ "${tests[$index]}" ]]; then
        echo ${messages[$index]}
        break
    fi
    sleep 5
    done
done
