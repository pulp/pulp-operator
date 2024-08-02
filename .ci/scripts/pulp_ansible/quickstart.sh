# This script will execute the component scripts and ensure that the documented examples
# work as expected.

# From the _scripts directory, run with `source quickstart.sh` (source to preserve the environment
# variables)
source setup.sh

echo "Role - Workflows"
source repo.sh
source distribution_repo.sh
source distribution_repo_version.sh
source remote.sh
source sync.sh

echo "Collection - Workflows"
source remote-collection.sh
source sync-collection.sh
