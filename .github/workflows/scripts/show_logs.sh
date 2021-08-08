#!/bin/bash -e
#!/usr/bin/env bash

echo ::group::OPERATOR_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-operator -c pulp-operator --tail=10000
echo ::endgroup::

echo ::group::PULP_API_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-api --tail=10000
echo ::endgroup::

echo ::group::PULP_CONTENT_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-content --tail=10000
echo ::endgroup::

echo ::group::PULP_WORKER_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-worker --tail=10000
echo ::endgroup::

echo ::group::PULP_RESOURCE_MANAGER_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-resource-manager --tail=10000
echo ::endgroup::

echo ::group::PULP_WEB_LOGS
sudo -E kubectl logs -l app.kubernetes.io/name=pulp-web --tail=10000
echo ::endgroup::
