# Getting started

## pulp-insta-demo.sh

[A script](https://raw.githubusercontent.com/pulp/pulp-operator/master/insta-demo/pulp-insta-demo.sh)
to install Pulp 3 on Linux systems with as many plugins as possible and an uninstaller.

Works by installing [K3s (lightweight kubernetes)](https://k3s.io/), and then deploying
pulp-operator on top of it.

Is not considered production ready because pulp-operator is not yet, it hides every config option,
and upgrades are not considered. Only suitable as a quick way to evaluate Pulp for the time
being.
