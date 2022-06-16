#!/usr/bin/env bash

# TODO: make a function that takes in files and replacements

# -- General replaces section

replacements=(
    RELATED_IMAGE_PULP:RELATED_IMAGE_HUB
    Pulp:AutomationHub
    "AutomationHub 3":AutomationHub
    pulp_pulpproject_org_pulp:automationhub_ansible_com_automationhub
    pulp.pulpproject.org:automationhub.ansible.com
)

# Replace in roles files; settings configmap is intentionally left out to keep settings in tact
for row in "${replacements[@]}"; do
    upstream="$(echo $row | cut -d: -f1)";
    downstream="$(echo $row | cut -d: -f2)";
    find ./roles -type f -name '*' \
      -not -path '*.md' \
	    -exec sed -i -e "s/${upstream}/${downstream}/g" {} \;
done

# Replace in watches.yaml
for row in "${replacements[@]}"; do
    upstream="$(echo $row | cut -d: -f1)";
    downstream="$(echo $row | cut -d: -f2)";
    sed -i -e "s/${upstream}/${downstream}/g" ./watches.yaml ;
done

# -- Replace deployment_type

for row in "${replacements[@]}"; do
    upstream="$(echo $row | cut -d: -f1)";
    downstream="$(echo $row | cut -d: -f2)";
    sed -i -e "s/pulp/automationhub/g" ./roles/backup/vars/main.yml \
                                       ./roles/backup/templates/event.yaml.j2 \
                                       ./roles/pulp-worker/defaults/main.yml \
                                       ./roles/postgres/defaults/main.yml \
                                       ./roles/restore/vars/main.yml ;
done

# Replace in manifest files

replacements=(
    RELATED_IMAGE_PULP:RELATED_IMAGE_HUB
)

for row in "${replacements[@]}"; do
    upstream="$(echo $row | cut -d: -f1)";
    downstream="$(echo $row | cut -d: -f2)";
    sed -i -e "s/${upstream}/${downstream}/g" ./bundle/manifests/pulp-operator.clusterserviceversion.yaml \
                                              ./config/manifests/bases/pulp-operator.clusterserviceversion.yaml \
                                              ./config/manager/manager.yaml ;
done

# -- Replace Service Account name

# TODO: consider changing deployment_type to automation-hub instead of automationhub

files=(
    bundle/manifests/pulp-operator.clusterserviceversion.yaml
    config/manifests/bases/pulp-operator.clusterserviceversion.yaml
)

for file in "${files[@]}"; do
   sed -i -e "s/pulp-operator-sa/automationhub-operator-sa/g" ${file};
done

# -- Replace pulp spec reference (based on manifest file name)

sed -i -e "s/pulp_pulpproject_org_pulp/automationhub_ansible_com_automationhub/g" ./playbooks/pulp.yml

# -- Inject cluster permissions for SA's and roles

# Or find a way to do this upstream
# https://code.engineering.redhat.com/gerrit/gitweb?p=automationhub-operator.git;a=blobdiff;f=bundle/manifests/pulp-operator.clusterserviceversion.yaml;h=b563c1c667824eab6b4feb70d798456841517328;hp=c2a54443eb6352e19e6f5e1071df8c9f5e93ad03;hb=5f5b65b9c32c38c1134f17dae9f1c68144f0f896;hpb=205aafd69999389f54b1a84537fe9f42550c9515

# -- Swap out postgres data path

sed -i -e "s/\/var\/lib\/postgresql/\/var\/lib\/pgsql/g" ./roles/postgres/defaults/main.yml

# -- Set default ingress_type to Route

files=(
    roles/pulp-api/defaults/main.yml
    roles/pulp-content/defaults/main.yml
    roles/pulp-status/defaults/main.yml
    roles/pulp-web/defaults/main.yml
)

for file in "${files[@]}"; do
    sed -i -e "s/ingress_type:\ none/ingress_type:\ Route/g" ${file};
done

# -- Set Fully Qualified Domain Names for k8s modules

# TODO


# TODO: inject `default: Route` on the following files:
  # bundle/manifests/pulp.pulpproject.org_pulps.yaml
  # config/crd/bases/pulpproject_v1beta1_pulp_crd.yaml

# TODO: Set this as a 'suggested' setting via alm-examples in a way that it gets used in the AAP wrapped operator

# Uneeded because it is still upstream
