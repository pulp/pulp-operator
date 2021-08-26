# Getting started

## Instant demo

[A script](https://raw.githubusercontent.com/pulp/pulp-operator/master/insta-demo/pulp-insta-demo.sh)
to install Pulp 3 on Linux systems with as many plugins as possible and an uninstaller.

Works by installing [K3s (lightweight kubernetes)](https://k3s.io/), and then deploying
pulp-operator on top of it.

Is not considered production ready because pulp-operator is not yet, it hides every config option,
and upgrades are not considered. Only suitable as a quick way to evaluate Pulp for the time
being.


## OpenShift

Currently pulp-operator is not available in the OpenShift catalogue, so as a first step you must create a catalogue entry:

```yaml
# pulp-catalog-source.yaml
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
 name: my-pulp-catalog
 namespace: openshift-marketplace
spec:
 sourceType: grpc
 image: quay.io/pulp/pulp-index:0.3.0

```

* Refer to [Getting started with the OpenShift CLI](https://docs.openshift.com/container-platform/4.7/cli_reference/openshift_cli/getting-started-cli.html)

* Verify the desired tag for `pulp-index` image [here](https://quay.io/repository/pulp/pulp-index?tab=tags)
```console
oc apply -f pulp-catalog-source.yaml
```

Wait a few seconds and refresh the OCP page. After the refresh, you can see a `my-pulp-catalog` entry
on the OperatorHub tab:
![OperatorHub tab](images/1.png "Pulp on OperatorHub tab")

Click `Pulp` and then `Install`:
![Installing pulp](images/2.png "Installing pulp operator")

![Installing pulp](images/3.png "Installing pulp operator")

Create a `Secret` with the `S3` credentials:
![S3 credentials Secret](images/4.png "S3 credentials Secret")

Click `Pulp`:
![Click on Pulp](images/5.png "Click on Pulp")

Select `S3` as the storage type and, on `S3 storage secret`, enter the name of the storage you created before,
e.g. `example-pulp-object-storage`:
![S3 credentials on Pulp kind](images/6.png "S3 credentials on Pulp kind")

Click `Advanced Configuration`,
select `Route` as Ingress type, fill in the `Route DNS host`, select `Edge` as Route TLS termination mechanism, and click on `Create`:
![Advanced Configuration](images/7.png "Advanced Configuration")

Wait a few minutes for pulp-operator to be successfully deployed!

You can check your `password` on `Secrets`, `example-pulp-admin-password`:
![Admin password Secret](images/8.png "Admin password Secret")

Verify your URL at `Networking > Routes`:
![Route URL](images/9.png "Route URL")

Use the URL from the previous step with `/pulp/api/v3/status`path and verify Pulp was successfully deployed:
![Pulp Status](images/10.png "Pulp Status")
