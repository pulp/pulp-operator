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
if [[ $(getenforce 2> /dev/null || echo "Disabled") != "Disabled" ]]; then
  if [[ ! -e /usr/sbin/semanage ]]; then
    if [ $FIXES = true ]; then
        set -x
        sudo dnf -y install /usr/sbin/semanage || sudo yum -y install /usr/sbin/semanage
        set +x
    else
      echo "SELinux is Enforcing or Permissive, but /usr/sbin/semanage is not installed."
      echo "k3s requires /usr/sbin/semanage to prevent SELinux errors."
      echo "Exiting."
    fi
  fi
fi

if [ $(basename `git rev-parse --show-toplevel`) != "pulp-operator" ]; then
  USER_REPO="pulp/pulp-operator"
  BRANCH="master"
  set -x
else
  set -x
  REMOTE_NAME=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} | cut -f 1 -d /)
  BRANCH=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} | cut -f 2- -d /)
  REMOTE=$(git remote get-url $REMOTE_NAME)
  # Processes examples of $REMOTE_NAME:
  # https://github.com/USERNAME/RE-PO_SITORY.git
  # git@github.com:USERNAME/RE-PO_SITORY.git
  USER_REPO=$(echo $REMOTE | grep -oP '([\w\-]+)\/([\w\-]+)(?=.git)')
fi
curl -SsL https://github.com/$USER_REPO/archive/$BRANCH.tar.gz | tar -xz || failure_message
cd pulp-operator-$BRANCH || failure_message
sudo .travis/k3s-install.sh --insta-demo || failure_message
sudo TRAVIS=true ./up.sh || failure_message
.travis/pulp-operator-check-and-wait.sh || test $? = 100 || failure_message
set +x
echo "Pulp has been installed in insta-demo mode."
echo ""
echo "If you wish to uninstall, run:"
echo "$ sudo /usr/local/bin/k3s-uninstall.sh"
