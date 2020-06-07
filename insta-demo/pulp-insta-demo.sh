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


sudo .travis/k3s-install.sh --insta-demo || failure_message
sudo TRAVIS=true ./up.sh || failure_message
.travis/pulp-operator-check-and-wait.sh || test $? = 100 || failure_message
set +x
echo "Pulp has been installed in insta-demo mode."
echo ""
echo "If you wish to uninstall, run:"
echo "$ sudo /usr/local/bin/k3s-uninstall.sh"
