#!/usr/bin/env bash

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
                                        roles/backup/templates/event.yaml.j2 \
                                        roles/pulp-worker/defaults/main.yml \
                                        roles/postgres/defaults/main.yml \
                                        roles/restore/vars/main.yml ;
done


# -- Swap out postgres data path

sed -i -e "s/\/var\/lib\/postgresql/\/var\/lib\/pgsql/g" ./roles/postgres/defaults/main.yml

# -- Inject RELATED_IMAGES_ references

# Uneeded because it is still upstream
