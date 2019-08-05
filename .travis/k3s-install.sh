#!/usr/bin/env bash
# coding=utf-8

# k3s-install.sh
# This script installs k3s (lightweight Kubernetes (K8s) and does anything else
# to setup the single-node k3s infrastructure for running Pulp containers in
# Travis CI.

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
curl -sfL https://get.k3s.io | sudo INSTALL_K3S_EXEC="--docker --kube-apiserver-arg service-node-port-range=80-32767" sh -
status=$(sudo systemctl status --full --lines=200 k3s)
if ! [[ $? ]] ; then
  echo "${status}"
  echo "SYSTEMD UNIT:"
  sudo cat /etc/systemd/system/k3s.service
  exit 1
fi
echo "k3s NODE Status:"
sudo kubectl get node

# By default, k3s lacks a storage class.
# https://github.com/rancher/k3s/issues/85#issuecomment-468293334
# This is the way to add a simple hostPath storage class.
sudo kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml
sudo kubectl get storageclass
# How make it the default StorageClass
sudo kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
sudo kubectl get storageclass

echo "NAT"
sudo iptables -L -t nat
echo "IPTABLES"
sudo iptables -L
echo "UFW"
sudo ufw status verbose
echo "CLUSTER-INFO"
sudo kubectl cluster-info
