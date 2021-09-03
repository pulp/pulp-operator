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

echo "Starting molecule test"
molecule -v test -s kind --destroy never
