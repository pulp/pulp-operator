# Pulp Operator

## Created with (based on template):
` ~/go/bin/operator-sdk new pulp-operator --api-version=example.com/v1alpha1 --kind=Pulp --type=ansible --generate-playbook`

## Built/pushed with:
`operator-sdk build --image-builder=buildah quay.io/mikedep333/pulp-operator:latest`

`podman push quay.io/mikedep333/pulp-operator:latest`

## Usage
`./up.sh`

`minikube service list`
