#!/bin/bash

# TODO: Check if all of these are needed.
# TODO: Check if these should only ever be run once; or require
# special logic to update
kubectl create -f deploy/crds/pulpproject_v1alpha1_pulp_crd.yaml
kubectl create -f deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml
kubectl apply -f deploy/crds/pulpproject_v1alpha1_pulp_cr.yaml

kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/operator.yaml
