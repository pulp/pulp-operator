# Sync repository foo using remote cbar
pulp ansible repository sync --name "foo" --remote "cbar"

# Use the -b option to have the sync task complete in the background
# e.g. pulp -b ansible repository sync --name "foo" --remote "cbar"

# After the task is complete, it gives us a new repository version
# Inspecting new repository version
pulp ansible repository version show --repository "foo"
