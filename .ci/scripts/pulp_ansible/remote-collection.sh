# Create a remote that syncs some versions of django into your repository.
pulp ansible remote -t "collection" create \
    --name "cbar" \
    --url "https://galaxy.ansible.com/" \
    --requirements $'collections:\n  - ansible.galaxy_collection'
# If requirements are in a file
# you can use the option '--requirements @<file_name>' instead.
