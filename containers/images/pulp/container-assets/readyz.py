#!/usr/bin/env python3

import os
import sys
import requests
import django
import socket
from datetime import timedelta


def is_api_healthy():
    """
    Checks if API is healthy
    """
    STATUS_PATH = sys.argv[1]
    print(f"Readiness probe checking {STATUS_PATH}")
    response = requests.get(
        f"http://localhost:24817{STATUS_PATH}", allow_redirects=True
    )
    data = response.json()

    if not data["online_workers"]:
        sys.exit(1)

    if not data["online_content_apps"]:
        sys.exit(2)

    if not data["database_connection"]["connected"]:
        sys.exit(3)

    if os.getenv("REDIS_SERVICE_HOST") and not data["redis_connection"]["connected"]:
        sys.exit(4)

    sys.exit(0)


def is_content_healthy():
    """
    Checks if Content is healthy
    """
    print(f"Readiness probe checking {socket.gethostname()}")
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "pulpcore.app.settings")
    django.setup()
    from django.utils import timezone
    from pulpcore.app.models.status import ContentAppStatus

    age_threshold = timezone.now() - timedelta(seconds=30)
    count = ContentAppStatus.objects.filter(
        name__endswith=socket.gethostname(), last_heartbeat__gte=age_threshold
    ).count()

    if not count:
        sys.exit(1)

    sys.exit(0)


def is_worker_healthy():
    """
    Checks if Worker is healthy
    """
    print(f"Readiness probe checking {socket.getfqdn()}")
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "pulpcore.app.settings")
    django.setup()
    from django.utils import timezone
    from pulpcore.app.models.task import Worker

    age_threshold = timezone.now() - timedelta(seconds=30)
    count = Worker.objects.filter(
        name__endswith=socket.getfqdn(), last_heartbeat__gte=age_threshold
    ).count()

    if not count:
        sys.exit(1)

    sys.exit(0)


if os.getenv("PULP_API_WORKERS"):
    is_api_healthy()

elif os.getenv("PULP_CONTENT_WORKERS"):
    is_content_healthy()

elif "-worker-" in socket.getfqdn():
    is_worker_healthy()
