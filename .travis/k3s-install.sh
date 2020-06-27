#!/usr/bin/env bash
# coding=utf-8

# k3s-install.sh
# This script installs k3s (lightweight Kubernetes (K8s) and does anything else
# to setup the single-node k3s infrastructure for running Pulp containers in
# Travis CI.

set -e

# This is their convenience installer script.
# Does a bunch of stuff, such as setting up a `kubectl` -> `k3s kubectl` symlink.
#
# It will install the latest version.
# We can always pass args to use a specific version or change other options.
#
# We want to allow devs to use Pulp's ports, like 80, 24816 or 24817,
# not just the default 30000-32767.
#
# TODO: Fix access to registry.centos.org
# https://github.com/rancher/k3s/issues/145#issuecomment-490143506
#
# Docker is used by CI (so that docker build can build images that k3s uses).
# Bundled containerd is used by pulp-insta-demo.sh for simplicity of install.
if [ "$1" = "--insta-demo" ]; then
  curl -sfL https://get.k3s.io | sudo INSTALL_K3S_EXEC="--kube-apiserver-arg service-node-port-range=80-32767" sh -
else
  curl -sfL https://get.k3s.io | sudo INSTALL_K3S_EXEC="--docker --kube-apiserver-arg service-node-port-range=80-32767" sh -
fi
status=$(sudo systemctl status --full --lines=200 k3s)
if ! [[ $? ]] ; then
  echo "${status}"
  echo "SYSTEMD UNIT:"
  sudo cat /etc/systemd/system/k3s.service
  exit 1
fi

if [ -x /usr/local/bin/kubectl ]; then
  KUBECTL=/usr/local/bin/kubectl
elif which kubectl > /dev/null; then
  KUBECTL=$(which kubectl)
  mkdir -p $HOME/.kube
  sudo cp -nf /etc/rancher/k3s/k3s.yaml $HOME/.kube/config
elif command -v kubectl > /dev/null; then
  KUBECTL=$(command -v kubectl)
else
    echo "$0: ERROR 1: Cannot find kubectl"
fi

# Wait for k3s being up without requiring netcat to be installed; pure bash.
timeout 120 bash -c 'until echo 2> /dev/null > /dev/tcp/localhost/6443; do sleep .2; done'

# Apparently that wait is still not enough. I suspect that the backend
# containers are not running yet.
echo "CLUSTER-INFO"
until sudo $KUBECTL cluster-info | grep "running at"; do sleep 1; done
echo "k3s NODE Status:"
# If no nodes are found, rc is still 0. So we cannot check that.
# The full latter string would be like:
# The connection to the server localhost:6443 was refused - did you specify the right host or port?
until sudo $KUBECTL get node -o wide 2>&1 | grep -v -E "No resources found.|The connection to the server"; do sleep 1; done

# By default, k3s lacks a storage class.
# https://github.com/rancher/k3s/issues/85#issuecomment-468293334
# This is the way to add a simple hostPath storage class.
sudo $KUBECTL apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml
if [ "$1" != "--insta-demo" ]; then
  sudo $KUBECTL get storageclass
fi
# How make it the default StorageClass
sudo $KUBECTL patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
if [ "$1" != "--insta-demo" ]; then
  sudo $KUBECTL get storageclass
fi

if [ "$1" != "--insta-demo" ]; then
  echo "NAT"
  sudo iptables -L -t nat
  echo "IPTABLES"
  sudo iptables -L
  echo "UFW"
  sudo ufw status verbose
fi
