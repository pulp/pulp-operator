Changelog
=========

<!---
    You should *NOT* be adding new change log entries to this file, this
    file is managed by towncrier. You *may* edit previous change logs to
    fix problems like typo corrections or such.
    To add a new change log entry, please see
    https://docs.pulpproject.org/contributing/git.html#changelog-update

    WARNING: Don't drop the next directive!
--->

<!-- TOWNCRIER -->

1.0.0-alpha.9 (2023-08-21)
==========================


Features
--------

- Added a check for missing file_storage_storage_class definition whenever
  file_storage_size or file_storage_access_mode is/are provided.
  [#946](https://github.com/pulp/pulp-operator/issues/946)
- Moved API container entrypoint migration script to k8s jobs.
  [#991](https://github.com/pulp/pulp-operator/issues/991)
- Added the OpenTelemetry support as sidecar container for pulp-api pods.
  [#1006](https://github.com/pulp/pulp-operator/issues/1006)
- Added support to define Redis PVC storage size.
  [#1016](https://github.com/pulp/pulp-operator/issues/1016)
- Added new fields to set resources for init-container and metrics sidecar containers.
  [#1019](https://github.com/pulp/pulp-operator/issues/1019)
- Added the `pulp_secret_key` field to set the Django `SECRET_KEY`.
  [#1040](https://github.com/pulp/pulp-operator/issues/1040)


Bugfixes
--------

- Fixed an issue in OCP clusters where every ingress would be created with the same configurations (regardless of ingressclass).
  [#917](https://github.com/pulp/pulp-operator/issues/917)
- Fixed an issue in OCP clusters where the "pulp-redirect" Ingress would not get removed after modifying ingress_class_name.
  [#918](https://github.com/pulp/pulp-operator/issues/918)
- Fixed an issue in `Ingress.spec.rules.http.paths` from non "nginx" or "openshift-default" ingresses.
  [#923](https://github.com/pulp/pulp-operator/issues/923)
- Modified the format of backup dir names.
  [#937](https://github.com/pulp/pulp-operator/issues/937)
- Fixed a bug that caused the CONTENT_ORIGIN scheme to always be https.
  [#1048](https://github.com/pulp/pulp-operator/issues/1048)


Improved Documentation
----------------------

- Added a doc section with instructions to install pulp-operator using Helm.
  [#1008](https://github.com/pulp/pulp-operator/issues/1008)


Deprecations and Removals
-------------------------

- The operator will not get the default ingress domain nor verify the ingressclass anymore to avoid the need of clusterroles.
  [#885](https://github.com/pulp/pulp-operator/issues/885)


Misc
----

- [#884](https://github.com/pulp/pulp-operator/issues/884), [#984](https://github.com/pulp/pulp-operator/issues/984), [#986](https://github.com/pulp/pulp-operator/issues/986), [#993](https://github.com/pulp/pulp-operator/issues/993), [#1041](https://github.com/pulp/pulp-operator/issues/1041)


----


1.0.0-alpha.8 (2023-06-23)
==========================


Bugfixes
--------

- Modified the default readiness probe endpoint when DOMAIN is enabled.
  [#987](https://github.com/pulp/pulp-operator/issues/987)


----


1.0.0-alpha.7 (2023-06-22)
==========================


Features
--------

- Modified the reconciliation for `pulpcore-content` to wait for `API` pods get
  into a READY state before updating the `Deployment` in case of image version change.
  [#969](https://github.com/pulp/pulp-operator/issues/969)
- Added a log message when restarting `api` and `content` pods in case of a
  secret reconciliation.
  [#973](https://github.com/pulp/pulp-operator/issues/973)


Bugfixes
--------

- Added a watcher on some secrets not managed by the operator and added a
  reconciliation loop in case these secrets get modified.
  [#521](https://github.com/pulp/pulp-operator/issues/521)


Improved Documentation
----------------------

- Added a networking section in configuration doc.
  [#666](https://github.com/pulp/pulp-operator/issues/666)
- Added more information regarding the usage and limitation of `emptyDir`.
  [#824](https://github.com/pulp/pulp-operator/issues/824)


----


1.0.0-alpha.6 (2023-04-27)
==========================


Bugfixes
--------

- The container_token_secret was not getting its name from Pulp CR.
  [#852](https://github.com/pulp/pulp-operator/issues/852)


Improved Documentation
----------------------

- Add Documentation for custom S3 endpoints
  [#882](https://github.com/pulp/pulp-operator/issues/882)


Misc
----

- [#858](https://github.com/pulp/pulp-operator/issues/858), [#935](https://github.com/pulp/pulp-operator/issues/935), [#942](https://github.com/pulp/pulp-operator/issues/942)


----


1.0.0-alpha.5 (2023-01-03)
==========================


Features
--------

- Added a feature to deploy and sync Galaxy execution environments.
  [#821](https://github.com/pulp/pulp-operator/issues/821)
- Modified postgres mount point to keep compatibility with ansible-based operator version.
  [#848](https://github.com/pulp/pulp-operator/issues/848)


Bugfixes
--------

- Added a check for `ingress_host` being null when `ingress_type` defined as "ingress".
  [#675](https://github.com/pulp/pulp-operator/issues/675)
- Fixed a permission/ownership error during bkp/restore procedure.
  [#808](https://github.com/pulp/pulp-operator/issues/808)
- Fixed a deadlock on status update.
  [#829](https://github.com/pulp/pulp-operator/issues/829)
- Fixed an issue on rendering Pulp settings wrongly.
  [#830](https://github.com/pulp/pulp-operator/issues/830)
- Fixed an issue with container token pub key mount point.
  [#834](https://github.com/pulp/pulp-operator/issues/834)
- Fixed an issue with default values for TOKEN_SERVER and TOKEN_AUTH_DISABLED in settings.py.
  [#836](https://github.com/pulp/pulp-operator/issues/836)


Improved Documentation
----------------------

- Added steps to configure and run backup/restore procedure.
  [#765](https://github.com/pulp/pulp-operator/issues/765)
- Added steps to manually configure ingress.
  [#771](https://github.com/pulp/pulp-operator/issues/771)
- Document how to install multiple instances of Pulp operator.
  [#827](https://github.com/pulp/pulp-operator/issues/827)


----


1.0.0-alpha.4 (2022-11-28)
==========================


Features
--------

- Added a field to set IngressClass name.
  [#674](https://github.com/pulp/pulp-operator/issues/674)
- Added a field to pass a secret name to configure route custom certificates.
  [#800](https://github.com/pulp/pulp-operator/issues/800)


Bugfixes
--------

- Fixed an issue with envtest failing because of an assessment with old value.
  [#807](https://github.com/pulp/pulp-operator/issues/807)


Improved Documentation
----------------------

- Described the Operator unmanaged state.
  [#792](https://github.com/pulp/pulp-operator/issues/792)


Misc
----

- [#796](https://github.com/pulp/pulp-operator/issues/796)


----


1.0.0-alpha.3 (2022-11-17)
==========================


Features
--------

- Added a configmap to avoid pulprestore controller execution.
  [#550](https://github.com/pulp/pulp-operator/issues/550)
- Add Ingress TLS secret
  [#676](https://github.com/pulp/pulp-operator/issues/676)
- Added a field to set affinity for bkp-manager pods.
  [#782](https://github.com/pulp/pulp-operator/issues/782)


Bugfixes
--------

- Make web available when ingress isn't nginx
  [#770](https://github.com/pulp/pulp-operator/issues/770)


----


1.0.0-alpha.2 (2022-11-09)
==========================


Bugfixes
--------

- Ensure reconciliation when ingress is modified
  [#672](https://github.com/pulp/pulp-operator/issues/672)
- Fixed an issue with .status.conditions[] not getting updated for pulpcore-workers.
  [#735](https://github.com/pulp/pulp-operator/issues/735)
- Fixed an issue with .status.conditions[] getting updated in a specific order.
  [#736](https://github.com/pulp/pulp-operator/issues/736)
- Fixed an issue in RequeueAfter reconciliation logic.
  [#747](https://github.com/pulp/pulp-operator/issues/747)
- Added a "retry" in case controller fails to update operator's status.conditions[].
  [#751](https://github.com/pulp/pulp-operator/issues/751)
- Fix ingress type assertion
  [#755](https://github.com/pulp/pulp-operator/issues/755)
- Set update error message as DEBUG instead of ERROR.
  [#756](https://github.com/pulp/pulp-operator/issues/756)


Misc
----

- [#763](https://github.com/pulp/pulp-operator/issues/763)


----


1.0.0-alpha.1 (2022-11-03)
==========================


Features
--------

- Added PDB configuration through Pulp CR.
  [#433](https://github.com/pulp/pulp-operator/issues/433)
- Modified affinity field to allow inter-pod affinity/anti-affinity configuration.
  [#434](https://github.com/pulp/pulp-operator/issues/434)
- Added option to mount custom CA.
  [#513](https://github.com/pulp/pulp-operator/issues/513)
- Added probe fields in pulp CR.
  [#516](https://github.com/pulp/pulp-operator/issues/516)
- Added configuration to change the operator log level.
  [#571](https://github.com/pulp/pulp-operator/issues/571)
- Added a field to control the restore deployment replicas.
  By default it will be set to false (restore controller will redeploy only a single replica of each component).
  [#572](https://github.com/pulp/pulp-operator/issues/572)
- Added more node selector configuration (cache and web pods).
  Added field to define route labels.
  [#577](https://github.com/pulp/pulp-operator/issues/577)
- Added default readiness probe for pulp-web pods.
  [#579](https://github.com/pulp/pulp-operator/issues/579)
- Added configuration to use external Redis instance.
  [#614](https://github.com/pulp/pulp-operator/issues/614)
- Modified (through processPodSecurityContext) the UID that runs the entrypoint of the container process.
  [#627](https://github.com/pulp/pulp-operator/issues/627)
- Modified Pulp CRD to collect info to connect to an external database from a Secret.
  [#630](https://github.com/pulp/pulp-operator/issues/630)
- Added a field to configure the deployment strategy.
  [#635](https://github.com/pulp/pulp-operator/issues/635)
- Let the operator namespace-scoped.
  [#657](https://github.com/pulp/pulp-operator/issues/657)
- Use Nginx Ingress as reverse proxy
  [#660](https://github.com/pulp/pulp-operator/issues/660)
- Added a check for configurations in non-ocp env with ingress_type==route.
  [#669](https://github.com/pulp/pulp-operator/issues/669)
- Updated CRD field comments.
  [#711](https://github.com/pulp/pulp-operator/issues/711)
- Utilize the renamed `pulp-minimal` and `galaxy-minimal` images. Also have CI test the new big s6-contining images `pulp` and `pulp-galaxy-ng`.
  [#717](https://github.com/pulp/pulp-operator/issues/717)
- Set nginx fields default values in controller (not in CR).
  [#722](https://github.com/pulp/pulp-operator/issues/722)
- Improved route paths provisioning loop.
  [#729](https://github.com/pulp/pulp-operator/issues/729)


Bugfixes
--------

- Added logic on how to handle different/multiple types of storage in Pulp CR.
  [#526](https://github.com/pulp/pulp-operator/issues/526)
- Fixed an issue with backup of PVCs manually created.
  [#580](https://github.com/pulp/pulp-operator/issues/580)
- Fixed an issue with backup controller failing when there was no signing secret.
  [#581](https://github.com/pulp/pulp-operator/issues/581)
- Fixed .status.condition not reflecting the real state.
  [#600](https://github.com/pulp/pulp-operator/issues/600)
- Add serviceaccounts permission
  [#601](https://github.com/pulp/pulp-operator/issues/601)
- Removed default values for Pulp database when configuring external PostgreSQL.
  [#622](https://github.com/pulp/pulp-operator/issues/622)
- Set ContainerTokenSecret as immutable (the controller will reconcile with the same value if the field is modified).
  Set AdminPasswordSecret as immutable (the controller will reconcile with the same value if the field is modified).
  Added ImagePullSecrets reconciliation logic.
  Fixed TrustedCa volumeMount reconciliation logic.
  Fixed NodeSelector reconciliation logic.
  Fixed Tolerations reconciliation logic.
  Fixed TopologySpreadConstraints reconciliation logic.
  Fixed ResourceRequirements removal logic.
  Fixed PDB removal logic.
  Fixed Strategy removal logic.
  Set Cache.ExternalCacheSecret as immutable (the controller will reconcile with the same value if the field is modified).
  Fixed Cache.RedisPort reconciliation logic.
  Fixed Cache.Resources reconciliation logic.
  Fixed Cache.NodeSelector reconciliation logic.
  Fixed Cache.Tolerations reconciliation logic.
  [#646](https://github.com/pulp/pulp-operator/issues/646)
- Fixed a bug in route reconciliation.
  [#648](https://github.com/pulp/pulp-operator/issues/648)
- Fixed the backoff loop not incrementing exponentially on error.
  [#650](https://github.com/pulp/pulp-operator/issues/650)
- Ensure Nginx Ingress Controller is used when multiple controllers are installed
  [#673](https://github.com/pulp/pulp-operator/issues/673)
- Added ingressclass clusterrole.
  [#709](https://github.com/pulp/pulp-operator/issues/709)
- Ensure ingress status conditions
  [#714](https://github.com/pulp/pulp-operator/issues/714)
- Fixed issue with headless services propagating new address to pulp-web pods.
  [#737](https://github.com/pulp/pulp-operator/issues/737)


Improved Documentation
----------------------

- Added steps to configure object storage.
  [#593](https://github.com/pulp/pulp-operator/issues/593)
- Added troubleshooting section.
  [#596](https://github.com/pulp/pulp-operator/issues/596)
- Stacktrace enabled only for above "panic" level.
  [#605](https://github.com/pulp/pulp-operator/issues/605)
- Added steps to configure operator's database.
  [#619](https://github.com/pulp/pulp-operator/issues/619)
- Fix broken links
  [#681](https://github.com/pulp/pulp-operator/issues/681)
- Added a section explaining default secrets created by the operator.
  [#683](https://github.com/pulp/pulp-operator/issues/683)


Misc
----

- [#678](https://github.com/pulp/pulp-operator/issues/678), [#692](https://github.com/pulp/pulp-operator/issues/692)


----


0.14.0 (2022-09-19)
===================


Features
--------

- Omitted pulp-web role if ingress_type==route, which brings some benefits like:
  * reduce point of failure
  * reduce complexity
  * reduce resource consumption
  * reduce communication hops
  [#436](https://github.com/pulp/pulp-operator/issues/436)
- Add support for pulp_container signing service
  [#564](https://github.com/pulp/pulp-operator/issues/564)


Bugfixes
--------

- Adding NodeSelector/Toleration to Redis Deployment
  [#561](https://github.com/pulp/pulp-operator/issues/561)
- Allows users to correctly set predefined pvc with backup_pvc.  It was hardcoded in the remove ownerReferences task.  Now correctly uses the dynamic variable backup_claim.
  [#610](https://github.com/pulp/pulp-operator/issues/610)


Misc
----

- [#489](https://github.com/pulp/pulp-operator/issues/489), [#590](https://github.com/pulp/pulp-operator/issues/590)


----


0.13.0 (2022-07-04)
===================


Features
--------

- Added more information on `.status.conditions` CR field.
  [#435](https://github.com/pulp/pulp-operator/issues/435)
- Added readiness probe to content and workers
  [#455](https://github.com/pulp/pulp-operator/issues/455)


Bugfixes
--------

- Remove ownerReferences from DB fields encryption secret to avoid garbage collection
  [#467](https://github.com/pulp/pulp-operator/issues/467)


Misc
----

- [#461](https://github.com/pulp/pulp-operator/issues/461), [#466](https://github.com/pulp/pulp-operator/issues/466)


----


0.12.0 (2022-06-15)
===================


Features
--------

- Make no_log configurable
  [#443](https://github.com/pulp/pulp-operator/issues/443)


Bugfixes
--------

- Improve pulp status health check
  [#447](https://github.com/pulp/pulp-operator/issues/447)


----


0.11.1 (2022-06-09)
===================


Bugfixes
--------

- Gunicorn API workers default to 2
  [#437](https://github.com/pulp/pulp-operator/issues/437)
- Ensure azure_connection_string is optional
  [#440](https://github.com/pulp/pulp-operator/issues/440)


----


0.11.0 (2022-06-02)
===================


Features
--------

- Upgrade to PostgreSQL 13 and add data migration logic
  [#358](https://github.com/pulp/pulp-operator/issues/358)
- Made Nginx, Gunicorn, HAproxy timeouts configurable
  [#418](https://github.com/pulp/pulp-operator/issues/418)
- The Pulp API can now be rerooted using the new ``API_ROOT`` setting. By default it is set to
  ``/pulp/``. Pulp appends the string ``api/v3/`` onto the value of ``API_ROOT``.
  [#421](https://github.com/pulp/pulp-operator/issues/421)


Bugfixes
--------

- Ensure Nginx `client_max_body_size` is correctly set
  [#418](https://github.com/pulp/pulp-operator/issues/418)
- Ensure content can be signed
  [#426](https://github.com/pulp/pulp-operator/issues/426)
- Fix restore when ``deployment_name`` is set
  [#427](https://github.com/pulp/pulp-operator/issues/427)


Misc
----

- [#407](https://github.com/pulp/pulp-operator/issues/407)


----


0.10.1 (2022-05-18)
===================


Bugfixes
--------

- Set reconcile period to 0s to resolve issue with reconciliation loop not converging
  [#385](https://github.com/pulp/pulp-operator/issues/385)
- Patch container-auth secret creation to ensure the reconciliation loop converges
  [#403](https://github.com/pulp/pulp-operator/issues/403)


Deprecations and Removals
-------------------------

- Revert #373 to ensure the reconciliation loop converges
  [#403](https://github.com/pulp/pulp-operator/issues/403)


----


0.10.0 (2022-05-12)
===================


Features
--------

- Add configurable timeout for pulp-api and pulp-content
  [#390](https://github.com/pulp/pulp-operator/issues/390)
- Add configurable workers for pulp-api and pulp-content
  [#392](https://github.com/pulp/pulp-operator/issues/392)


Bugfixes
--------

- Fix a reference to an incorrect variable in pulp-status role
  [#388](https://github.com/pulp/pulp-operator/issues/388)
- Provide default values for container registry
  [#394](https://github.com/pulp/pulp-operator/issues/394)


Misc
----

- [#386](https://github.com/pulp/pulp-operator/issues/386)


----


0.9.0 (2022-04-27)
==================

Features
--------

- Modified image_pull_secret to allow users to provide multiple secrets.
  [#343](https://github.com/pulp/pulp-operator/issues/343)
- Implement the galaxy collection signing service
  [#362](https://github.com/pulp/pulp-operator/issues/362)
- Backup & restore the default signing service
  [#366](https://github.com/pulp/pulp-operator/issues/366)
- Enable backup for ReadWriteOnce access mode
  [#380](https://github.com/pulp/pulp-operator/issues/380)


Bugfixes
--------

- Fix backup/restore events
  [#378](https://github.com/pulp/pulp-operator/issues/378)


Misc
----

- [#374](https://github.com/pulp/pulp-operator/issues/374)


----


0.8.0 (2022-03-14)
==================


Features
--------

- Add ability to configure extra args for postgres
  [#344](https://github.com/pulp/pulp-operator/issues/344)
- Add the ability to specify topologySpreadConstraints
  [#345](https://github.com/pulp/pulp-operator/issues/345)
- Allow service annotations not only for LoadBalancer type
  [#346](https://github.com/pulp/pulp-operator/issues/346)
- Support nodeSelector and tolerations
  [#348](https://github.com/pulp/pulp-operator/issues/348)


Bugfixes
--------

- Ensure the operator works with pre-defined TLS secret
  [#354](https://github.com/pulp/pulp-operator/issues/354)


----


0.7.1 (2022-02-22)
==================


Bugfixes
--------

- Made Redis optional when installing pulp
  [#323](https://github.com/pulp/pulp-operator/issues/323)
- Made Operator work with arbitrary namespaces
  [#326](https://github.com/pulp/pulp-operator/issues/326)
- Made web image and ingress to have the same max_body_size
  [#330](https://github.com/pulp/pulp-operator/issues/330)
- Fixed pulp-api and pulp-web liveness probes.
  [#332](https://github.com/pulp/pulp-operator/issues/332)
- Fixes TokenReview authentication
  [#337](https://github.com/pulp/pulp-operator/issues/337)


----


0.7.0 (2021-12-21)
==================


Features
--------

- Support cert-manager format on container token secret
  [#313](https://github.com/pulp/pulp-operator/issues/313)
- Enable Execution Environments by default
  [#315](https://github.com/pulp/pulp-operator/issues/315)


Bugfixes
--------

- Renamed services to avoid overwriting environment variables
  https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
  [#309](https://github.com/pulp/pulp-operator/issues/309)


----


0.6.1 (2021-12-09)
==================


Bugfixes
--------

- Mount `/var/lib/pulp/tmp` on pulp-content
  [#299](https://github.com/pulp/pulp-operator/issues/299)
- Raise resource limits for worker container to avoid OOMKill
  [#302](https://github.com/pulp/pulp-operator/issues/302)
- Raise resource limits for content container to avoid OOMKill
  [#303](https://github.com/pulp/pulp-operator/issues/303)


----


0.6.0 (2021-12-06)
==================


Bugfixes
--------

- Fix node affinity handling
  [#289](https://github.com/pulp/pulp-operator/issues/289)
- Fixed web containers initialization
  [#295](https://github.com/pulp/pulp-operator/issues/295)


----


0.5.0 (2021-11-05)
==================


Features
--------

- Made request size limit configurable
  [#227](https://github.com/pulp/pulp-operator/issues/227)
- Ensure resource manager is not started for pulpcore >= 3.16
  [#231](https://github.com/pulp/pulp-operator/issues/231)
- Set RELATED_IMAGE_ vars to enable disconnected deployments
  [#232](https://github.com/pulp/pulp-operator/issues/232)


Bugfixes
--------

- Image pull policy defaults to IfNotPresent
  [#229](https://github.com/pulp/pulp-operator/issues/229)


----


0.4.0 (2021-10-15)
==================


Features
--------

- Removed tags, registry, and projects so users can add images with custom registries and tags in image override
  [#218](https://github.com/pulp/pulp-operator/issues/218)
- Create or import a key for pulp-api to use when encrypting sensitive db fields
  [#8730](https://pulp.plan.io/issues/8730)
- Enable new tasking system
  [#9020](https://pulp.plan.io/issues/9020)
- Added support to override PosgreSQL sslmode
  [#9421](https://pulp.plan.io/issues/9421)


Bugfixes
--------

- Ensure default storage for Postgres
  [#221](https://github.com/pulp/pulp-operator/issues/221)


Deprecations and Removals
-------------------------

- Move from cluster-scoped operator model to namespace-scoped model
  [#208](https://github.com/pulp/pulp-operator/issues/208)
- Dropping OCP 4.6 support
  [#9330](https://pulp.plan.io/issues/9330)


Misc
----

- [#206](https://github.com/pulp/pulp-operator/issues/206), [#209](https://github.com/pulp/pulp-operator/issues/209), [#215](https://github.com/pulp/pulp-operator/issues/215), [#9217](https://pulp.plan.io/issues/9217)


----


0.3.0 (2021-07-14)
==================


Features
--------

- Enable container based database migration support
  [#8472](https://pulp.plan.io/issues/8472)
- Enable backup of database and secrets associated with Pulp custom resource
  [#8473](https://pulp.plan.io/issues/8473)
- Enable backup of storage associated with Pulp custom resource
  [#8474](https://pulp.plan.io/issues/8474)
- Enable restore of deployment associated with Pulp custom resource backup
  [#8513](https://pulp.plan.io/issues/8513)
- Add additional backup and restore flexibility to allow for restore from only a PVC
  [#8630](https://pulp.plan.io/issues/8630)
- Allow user to specify the storage class for the Redis PVC
  [#8877](https://pulp.plan.io/issues/8877)


Bugfixes
--------

- Allow user to specify empty string for PostgreSQL PVC storage class
  [#8733](https://pulp.plan.io/issues/8733)
- Update nodeport templating in API and Content services
  [#8810](https://pulp.plan.io/issues/8810)
- Fix collision on file_storage fact usage after pulp prefix cleanup
  [#8832](https://pulp.plan.io/issues/8832)
- Fix Nodeport flow to create ports in standard range and only on the web service. Also allows node_ip discover based on where the pod is running.
  [#8833](https://pulp.plan.io/issues/8833)
- Resolve Pulp status correctly when deployed in a separate namespace
  [#8880](https://pulp.plan.io/issues/8880)


Improved Documentation
----------------------

- Document how to deploy Pulp on OpenShift
  [#8836](https://pulp.plan.io/issues/8836)


Misc
----

- [#8530](https://pulp.plan.io/issues/8530), [#8563](https://pulp.plan.io/issues/8563), [#8598](https://pulp.plan.io/issues/8598)


----


0.2.0 (2021-03-26)
==================


Features
--------

- Add deployment of nginx webserver with pulp snippets
  [#5657](https://pulp.plan.io/issues/5657)
- Container building machinery for the operator
  [#7171](https://pulp.plan.io/issues/7171)
- Enable the creation of Ingress or Route objects based on the specifications within the custom resource
  [#8272](https://pulp.plan.io/issues/8272)
- Deploy postgres database using a secret to store configuration instead of it existing in the custom resource; allows credentials to be kept secret.
  [#8289](https://pulp.plan.io/issues/8289)
- Enable the use of S3 compliant or Azure object storage as storage backend
  [#8361](https://pulp.plan.io/issues/8361)
- Operator will provide information data via custom resource status object
  [#8402](https://pulp.plan.io/issues/8402)
- Enable installation of operator using OLM catalog
  [#8409](https://pulp.plan.io/issues/8409)
- Enable resource requirement specification for deployments and have operator check for running nodes and healthy status
  [#8456](https://pulp.plan.io/issues/8456)


Bugfixes
--------

- Only build plugins from pulp org
  [#7234](https://pulp.plan.io/issues/7234)
- Fix storage option check so that Azure Blob Storage can be used as a backend
  [#8424](https://pulp.plan.io/issues/8424)


Misc
----

- [#5134](https://pulp.plan.io/issues/5134), [#5141](https://pulp.plan.io/issues/5141), [#5142](https://pulp.plan.io/issues/5142), [#7107](https://pulp.plan.io/issues/7107), [#8273](https://pulp.plan.io/issues/8273), [#8293](https://pulp.plan.io/issues/8293), [#8294](https://pulp.plan.io/issues/8294), [#8345](https://pulp.plan.io/issues/8345), [#8353](https://pulp.plan.io/issues/8353), [#8370](https://pulp.plan.io/issues/8370), [#8378](https://pulp.plan.io/issues/8378), [#8399](https://pulp.plan.io/issues/8399), [#8418](https://pulp.plan.io/issues/8418)


----
