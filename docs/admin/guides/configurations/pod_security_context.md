# Overriding the pod security context

By default, Pulp Operator runs the `api`, `content`, `worker`, and `web` pods with
a hardcoded `runAsUser: 700` and `fsGroup: 700` [pod security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/).

This works for the images Pulp Operator ships by default, but a custom Pulp image
that does not use UID/GID 700 for its `pulp` user needs a different value. The
`pod_security_context` field lets you override the pod-level security attributes
per component instead of patching the operator.

!!! INFO

    `pod_security_context` only applies to the `api`, `content`, `worker` and `web`
    pods. It has no effect on:

    * the `database` and `cache` (Redis) pods;
    * the `AdminPasswordJob`, `MigrationJob` and `SigningJob` jobs;
    * `PulpBackup` and `PulpRestore` pods.

    These keep their own hardcoded values.

!!! INFO

    Setting `pod_security_context` **replaces** the whole default, it does not merge
    field by field. `pod_security_context: {fsGroup: 2000}` drops `runAsUser: 700`
    entirely rather than keeping it alongside the new `fsGroup`. Set every attribute
    you need explicitly.

To override the pod security context configure the following fields:

* `api.pod_security_context` [**optional**] overrides the pod-level `securityContext` for api pods. If not defined, `runAsUser` and `fsGroup` default to `700`.
* `content.pod_security_context` [**optional**] overrides the pod-level `securityContext` for content pods. If not defined, `runAsUser` and `fsGroup` default to `700`.
* `worker.pod_security_context` [**optional**] overrides the pod-level `securityContext` for worker pods. If not defined, `runAsUser` and `fsGroup` default to `700`.
* `web.pod_security_context` [**optional**] overrides the pod-level `securityContext` for web pods. If not defined, `runAsUser` and `fsGroup` default to `700` (or left unset on OpenShift, where the platform assigns a UID/GID automatically).

On OpenShift, api, content and worker pods also get an empty `securityContext` by
default (the platform's SCC assigns a UID/GID automatically) unless
`pod_security_context` is set, in which case your override is used instead.

Example of CR configuration to run a custom image under UID/GID 101:
```yaml
spec:
  api:
    pod_security_context:
      runAsUser: 101
      fsGroup: 101
  content:
    pod_security_context:
      runAsUser: 101
      fsGroup: 101
  worker:
    pod_security_context:
      runAsUser: 101
      fsGroup: 101
  web:
    pod_security_context:
      runAsUser: 101
      fsGroup: 101
```

The field accepts any Kubernetes [`PodSecurityContext`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#podsecuritycontext-v1-core)
attribute (for example `runAsGroup` or `fsGroupChangePolicy`), not just `runAsUser` and `fsGroup`.

!!! INFO

    Some container runtimes reject `runAsGroup` when `runAsUser` is not also
    set, with an error like `user group "N" is specified without user`.
    Set both together if you need to override the group.
