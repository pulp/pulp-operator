#!/bin/bash -e
#!/usr/bin/env bash

KIND_URL=https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64

if [[ "$OSTYPE" == "darwin"* ]]; then
  KIND_URL=https://kind.sigs.k8s.io/dl/v0.11.1/kind-darwin-amd64
fi

curl -Lo ./kind $KIND_URL
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

make kustomize
kustomize version

sed -i "s/ReadWriteMany/ReadWriteOnce/g" config/samples/pulpproject_v1beta1_pulp_cr.ci.yaml
find ./roles/*/templates/*.yaml.j2 -exec sed -i 's/pulp-operator-sa/osdk-sa/g' {} \;

echo "Starting molecule test"
molecule -v test -s kind --destroy never
