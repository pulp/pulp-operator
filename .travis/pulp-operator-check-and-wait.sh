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
  sudo kubectl get pvc
  sudo kubectl get pv
  df -h
  sudo kubectl -n local-path-storage get pod
  sudo kubectl -n local-path-storage logs $STORAGE_POD
}

# Once the services are both up, the pods will be in a Pending state.
# Before the services are both up, the pods may not exist at all.
# So check for the services being up 1st.
for tries in {0..30}; do
  services=$(sudo kubectl get services)
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
      echo "SERVICES:"
      echo "$services"
      storage_debug
      exit 2
    fi
  fi
  sleep 5
done   

# This needs to be down here. Otherwise, the storage pod may not be
# up in time.
STORAGE_POD=$(sudo kubectl -n local-path-storage get pod | awk '/local-path-provisioner/{print $1}')

# NOTE: Before the pods can be started, they must be downloaded/cached from
# quay.io .
# Therefore, this wait is highly dependent on network speed.
for tries in {0..180}; do
  pods=$(sudo kubectl get pods -o wide)
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
      echo "DOCKER IMAGE CACHE:"
      sudo docker images
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

for tries in {0..120}; do
  if [[ $tries -eq 120 ]]; then
    echo "ERROR 5: pulp-api never connected to the database"
    storage_debug
    echo "$output"
    exit 5
  fi
  output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
  if [[ "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .database_connection.connected)" = "true" ]]; then
    echo "pulp-api is connected to the database"
    break
  fi
  sleep 5
done

for tries in {0..120}; do
  if [[ $tries -eq 120 ]]; then
    echo "ERROR 6: pulp-api never connected to redis"
    storage_debug
    echo "$output"
    exit 6
  fi
  output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
  if [[ "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .redis_connection.connected)" = "true" ]]; then
    echo "pulp-api is connected to redis"
    break
  fi
  sleep 5
done

for tries in {0..120}; do
  if [[ $tries -eq 120 ]]; then
    echo "ERROR 7: Content app never came online"
    storage_debug
    echo "$output"
    exit 7
  fi
  output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
  if [[ "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .online_content_apps)" != "[]" ]]; then
    echo "Content app is online"
    break
  fi
  sleep 5
done

for tries in {0..120}; do
  if [[ $tries -eq 120 ]]; then
    echo "ERROR 8: Worker(s) never came online"
    storage_debug
    echo "$output"
    exit 8
  fi
  output=$(http --timeout 5 --check-status --pretty format --print hb $URL 2>&1)
  if [[ "$(echo "$output" | sed -ne '/{/,$ p' | jq -r .online_workers)" != "[]" ]]; then
    echo "1 or more worker is online"
    break
  fi
  sleep 5
done
