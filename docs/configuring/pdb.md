# Specifying a Disruption Budget

Pulp operator allows to configure a [PodDisruptionBudget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) for pulpcore components (`pulp-api`, `pulp-content`, `pulp-worker`, `pulp-web`).

!!! info
    Make sure you know what you are doing before configuring `PDB`.
    If not configured correctly it can cause unexpected behavior, like getting
    a node in a hang state during maintenance (node drain) or cluster upgrade.

It is possible to set only `maxUnavailable` or `minAvailable`. Trying to configure both for the same
PDB will fail operator execution.

The label selector will be handled by the operator based on Pulp CR spec.

For example, to configure API pods with PDB `minAvailable` and worker pods with `maxUnavailable`:
```yaml
$ oc edit pulp
...
spec:
  api:
    pdb:
      minAvailable: 1
  ...
  worker:
    pdb:
      maxUnavailable: 1
...
```