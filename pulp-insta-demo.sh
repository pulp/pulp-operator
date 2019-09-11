#!/bin/bash -xe

# pulp-insta-demo.sh
# This script quickly deploys pulp via k3s (lightweight Kubernetes), its
# embedded containerd, the pulp-operator (container).
#
# The pulp-operator in turn pulls multiple containers, including the
# all-n-one "pulp" container with several plugins.

FIXES=false
# Replace with getopts if we start adding more args
if [ "$1" = "--help" ] || [ "$1" == "-h" ]; then
  echo "Usage $0 [ -f | --fixes ]"
  exit 1
elif
  [ "$1" = "--fixes" ] || [ "$1" = "-f" ]; then
  FIXES=true
fi

# We do not want to workaround every single possible reason the script may fail,
# but our test environment (fedora30 vagrant box) needs this.
if [[ $(getenforce || echo "Disabled") != "Disabled" ]]; then
  if [[ ! -e /usr/sbin/semanage ]]; then
    if [ $FIXES = true ]; then
        sudo dnf -y install /usr/sbin/semanage || sudo yum -y install /usr/sbin/semanage
    else
      echo "SELinux is Enforcing or Permissive, but /usr/sbin/semanage is not installed."
      echo "k3s requires /usr/sbin/semanage to prevent SELinux errors."
      echo "Exiting."
    fi
  fi
fi

# TODO: Try-catch function in case these fail; advising users to recover or
# uninstall. Similar to show_logs_and_return_non_zero .
curl -SsL https://github.com/mikedep333/pulp-operator/archive/accomodate-insta-demo.tar.gz | tar -xz
cd pulp-operator-accomodate-insta-demo
sudo .travis/k3s-install.sh --insta-demo
sudo TRAVIS=true ./up.sh
.travis/pulp-operator-check-and-wait.sh || test $? = 100
echo "Pulp has been installed in insta-demo mode."
echo ""
echo "If you wish to uninstall, run:"
echo "$ sudo /usr/local/bin/k3s-uninstall.sh"
