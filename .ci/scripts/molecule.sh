#!/bin/bash -e
#!/usr/bin/env bash

if command -v KIND_URL > /dev/null; then
  echo "kind is installed already"
else
    KIND_URL=https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64

    if [[ "$OSTYPE" == "darwin"* ]]; then
      KIND_URL=https://kind.sigs.k8s.io/dl/v0.11.1/kind-darwin-amd64
    fi

    curl -Lo ./kind $KIND_URL
    chmod +x ./kind
    mv ./kind /usr/local/bin/kind

fi

make kustomize
kustomize version

find ./roles/*/templates/*.yaml.j2 -exec sed -i 's/{{ deployment_type }}-operator-sa/osdk-sa/g' {} \;

echo "Starting molecule test"
molecule -v test -s kind --destroy never
