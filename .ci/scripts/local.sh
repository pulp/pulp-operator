#!/usr/bin/env bash

kustomize build config/local | kubectl apply -f -
make manifests generate fmt vet CR_KIND=$1 CR_DOMAIN=$2 CR_PLURAL=$3 APP_IMAGE=$4 WEB_IMAGE=$5
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
Environment="DEV_MODE=true"
[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl enable pulp-operator.service --now
fi
