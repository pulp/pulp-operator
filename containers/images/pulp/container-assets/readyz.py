#!/usr/bin/env python3

import os
import sys
import requests
import socket


def is_api_healthy(path):
    """
    Checks if API is healthy
    """
    print(f"Readiness probe checking {path}")
    response = requests.get(
        f"http://localhost:24817{path}", allow_redirects=True
    )
    data = response.json()

    if not data["database_connection"]["connected"]:
        sys.exit(3)

    if os.getenv("REDIS_SERVICE_HOST") and not data["redis_connection"]["connected"]:
        sys.exit(4)

    sys.exit(0)


def is_content_healthy(path):
    """
    Checks if Content is healthy
    """
    print(f"Readiness probe checking {socket.gethostname()}")
    response = requests.head(f"http://localhost:24816{path}")
    response.raise_for_status()

    sys.exit(0)

if os.getenv("PULP_API_WORKERS"):
    is_api_healthy(sys.argv[1])

elif os.getenv("PULP_CONTENT_WORKERS"):
    is_content_healthy(sys.argv[1])
