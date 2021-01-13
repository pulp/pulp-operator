#!/bin/bash -e

# pulp-insta-demo.sh
# This script quickly deploys pulp via k3s (lightweight Kubernetes), its
# embedded containerd, the pulp-operator (container).
#
# The pulp-operator in turn pulls multiple containers, including the
# all-n-one "pulp" container with several plugins.

FIXES=false
if [ "$1" = "--help" ] || [ "$1" == "-h" ]; then
  echo "Usage $0 [ -f | --fixes ]"
  exit 1
elif
  [ "$1" = "--fixes" ] || [ "$1" = "-f" ]; then
  FIXES=true
fi

failure_message() {
  set +x
  echo "$0 failed to install."
  echo ""
  echo "You can either try to fix the errors and re-run it,"
  echo "or uninstall by running:"
  echo "$ sudo /usr/local/bin/k3s-uninstall.sh"
  exit 1
}

# Replace with getopts if we start adding more args
# We do not want to workaround every single possible reason the script may fail,
# but our test environment (fedora30 vagrant box) needs this.
if [[ $(getenforce 2> /dev/null) = "Enforcing" ]]; then
  if [[ ! -e /usr/sbin/semanage ]]; then
    if [ $FIXES = true ]; then
        set -x
        sudo dnf -y install /usr/sbin/semanage || sudo yum -y install /usr/sbin/semanage
        set +x
    else
      echo "SELinux is Enforcing, but /usr/sbin/semanage is not installed."
      echo "k3s requires /usr/sbin/semanage to prevent SELinux errors."
      echo "Exiting."
    fi
  fi
fi

# The behavior of this block is as follows:
# 1. If the user downloads this script directly, grab the pulp/pulp-operator
#    repo's master branch in a tarball from github. Few commands required.
# 2. If a developer is testing this on his machine or VM, use git to determine
#    the github user, repo and branch, and test the tarball download process
#    to simulate #1. This requires the developer to commit & push 1st.
# 3. If Travis, use Travis env vars (our git commands won't work without
#    branches), and test the tarball download process to simulate #1.
# 4. If in Vagrant (to test multiple distros), mount the directory.
#    This does not test the tarball download process unfortunately,
#    but Vagrant's shell provisioner only uploads the script, so we have
#    no good & easy-to-implement option.
if command -v git > /dev/null && [[ "$(basename `git rev-parse --show-toplevel`)" == "pulp-operator" ]]; then
  set -x
  REMOTE_NAME=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} | cut -f 1 -d /)
  # Travis does not checkout a branch, just a specific commit.
  if [ -n "${GITHUB_REF}" ]; then
    BRANCH=${GITHUB_REF##*/}
  else
    BRANCH=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} | cut -f 2- -d /)
  fi
  if [ -n "${GITHUB_REPOSITORY}" ]; then
    USER_REPO=$GITHUB_REPOSITORY
  else
    REMOTE=$(git remote get-url $REMOTE_NAME)
    # Processes examples of $REMOTE_NAME:
    # https://github.com/USERNAME/RE-PO_SITORY.git
    # git@github.com:USERNAME/RE-PO_SITORY.git
    USER_REPO=$(echo $REMOTE | grep -oP '([\w\-]+)\/([\w\-]+)(?=.git)')
  fi
else
  USER_REPO="pulp/pulp-operator"
  BRANCH="master"
  set -x
fi
URL=https://github.com/$USER_REPO/archive/$BRANCH.tar.gz
VAGRANT_MOUNT_DIR=/home/vagrant/devel/pulp-operator
if [ -e $VAGRANT_MOUNT_DIR ]; then
  echo "Vagrant Detected. Using $VAGRANT_MOUNT_DIR instead of"
  echo "$URL"
  cd $VAGRANT_MOUNT_DIR || failure_message
else
  curl -SsL $URL | tar -xz || failure_message
  cd pulp-operator-$BRANCH || failure_message
fi

echo "=================================== K3S Install ==================================="
sudo -E .ci/scripts/k3s-install.sh --insta-demo || failure_message
echo "=================================== K3S Up ==================================="
sudo -E ./up.sh || failure_message
echo "=================================== Check and wait ==================================="
echo ""
.ci/scripts/pulp-operator-check-and-wait.sh || test $? = 100 || failure_message
set +x
echo "Pulp has been installed in insta-demo mode."
echo ""
echo "If you wish to uninstall, run:"
echo "$ sudo /usr/local/bin/k3s-uninstall.sh"
