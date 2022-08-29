#!/usr/bin/env bash

kustomize build config/local | kubectl apply -f -
make manifests generate fmt vet
if [[ "$CI_TEST" == "true" ]] ; then
    make build
    sudo mv ./bin/manager /usr/local/bin/pulp
    cat << EOF | sudo tee /usr/lib/systemd/system/pulp-operator.service
[Unit]
Description=Pulp Operator
[Service]
WorkingDirectory=/usr/local/bin/
ExecStart=pulp
Restart=always
User=root
Environment="KUBECONFIG=$HOME/.kube/config"
[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl enable pulp-operator.service --now
fi
