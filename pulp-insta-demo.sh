#!/bin/bash -xe

# pulp-insta-demo.sh
# This script quickly deploys pulp via k3s (lightweight Kubernetes), its
# embedded containerd, the pulp-operator (container).
#
# The pulp-operator in turn pulls multiple containers, including the
# all-n-one "pulp" container with several plugins.

# We do not want to workaround every single possible reason the script may fail,
# but our test environment (fedora30 vagrant box) needs this.
if [[ $(getenforce || echo "Disabled") != "Disabled" ]]; then
  if [[ ! -e /usr/sbin/semanage ]]; then
    sudo dnf -y install /usr/sbin/semanage || sudo yum -y install /usr/sbin/semanage
  fi
fi
   
# This is their convenience installer script.
# Does a bunch of stuff, such as setting up a `kubectl` -> `k3s kubectl` symlink.
curl -sfL https://get.k3s.io | sudo INSTALL_K3S_EXEC="--kube-apiserver-arg service-node-port-range=80-32767" sh -
sleep 30
echo "k3s NODE Status:"
sudo kubectl get node

# By default, k3s lacks a storage class.
# https://github.com/rancher/k3s/issues/85#issuecomment-468293334
# This is the way to add a simple hostPath storage class.
sudo kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml
# How make it the default StorageClass
sudo kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
sudo kubectl cluster-info

curl -L https://github.com/pulp/pulp-operator/archive/master.tar.gz | tar -xz
cd pulp-operator-master
sudo ./up.sh
.travis/pulp-operator-check-and-wait.sh
