#!/bin/bash -e
#!/usr/bin/env bash

curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

make kustomize

kustomize version

echo "Starting molecule test"
molecule test -s kind
