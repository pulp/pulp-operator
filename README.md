# Pulp Operator

## Created with (based on template):
` ~/go/bin/operator-sdk new pulp-operator --api-version=pulpproject.org/v1alpha1 --kind=Pulp --type=ansible --generate-playbook`

## Built/pushed with:
`operator-sdk build --image-builder=buildah quay.io/pulp/pulp-operator:latest`

`podman login quay.io`

`podman push quay.io/pulp/pulp-operator:latest`

## Usage

Review `deploy/crds/pulpproject_v1alpha1_pulp_cr.default.yaml`. If the variables' default values are not correct for your environment, copy to `deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml`, uncomment "spec:", and uncomment & adjust the variables.

`./up.sh`

`minikube service list`

or

Get external ports:

`kubectl get services`

Get external IP addresses:

`kubectl get pods -o wide`
