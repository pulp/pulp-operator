#!/bin/bash

make bundle-check

echo "Comparing files ..."
diff -qr bundle /tmp/bundle
if [ $? != 0 ]; then
  echo """
There is probably an update made in the CRD that is pending in the above bundle file(s).
Please, run 'make bundle' and update the PR with the changes if necessary or double-check
if there is any misconfiguration in CRD, roles, manifests, etc from config dir.
  """
  exit 1
fi