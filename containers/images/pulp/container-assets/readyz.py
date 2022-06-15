#!/usr/bin/env python3

import os
import sys
import requests

STATUS_PATH = sys.argv[1]
print(f"Readiness probe checking {STATUS_PATH}")
response = requests.get(f"http://localhost:24817{STATUS_PATH}", allow_redirects=True)
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
