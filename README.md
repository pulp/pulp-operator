# Pulp Operator

## Created with (based on template):
` ~/go/bin/operator-sdk new pulp-operator --api-version=pulpproject.org/v1alpha1 --kind=Pulp --type=ansible --generate-playbook`

## Built/pushed with:
`operator-sdk build --image-builder=buildah quay.io/pulp/pulp-operator:latest`

`podman login quay.io`

`podman push quay.io/pulp/pulp-operator:latest`

## Usage

Review `deploy/pulp-operator.default.config-map.yml`. If the values are not correct for your environment, copy to `deploy/pulp-operator.config-map.yml` and adjust them.

`./up.sh`

`minikube service list`

or

Get external ports:

`kubectl get services`

Get external IP addresses:

`kubectl get pods -o wide`
