# Pulp Operator

## Created with (based on template):
` ~/go/bin/operator-sdk new pulp-operator --api-version=example.com/v1alpha1 --kind=Pulp --type=ansible`

## Built/pushed with:
`operator-sdk build --image-builder=buildah quay.io/mikedep333/pulp-operator:v0.0.1`

`podman push quay.io/mikedep333/pulp-operator:v0.0.1`

## Usage
kubectl create -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
