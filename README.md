# Pulp Operator

## Created with (based on template):
` ~/go/bin/operator-sdk new pulp-operator --api-version=pulpproject.org/v1alpha1 --kind=Pulp --type=ansible --generate-playbook`

## Built/pushed with:
`operator-sdk build --image-builder=buildah quay.io/pulp/pulp-operator:latest`

`podman login quay.io`

`podman push quay.io/pulp/pulp-operator:latest`

## Usage
`./up.sh`

`minikube service list`
