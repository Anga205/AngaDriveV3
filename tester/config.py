"""Connection configuration for the AngaDrive integration tests.

All values can be overridden via environment variables so the suite can be
pointed at any running instance (local, staging, or production).
"""

import os

# Host:port of the Go backend. Defaults to localhost:8080.
API_URL = os.getenv("API_URL", "localhost:8080")

# Websocket endpoint (the Go server exposes it at /ws).
WS_URL = f"ws://{API_URL}/ws"

# HTTP base URL (used for the chunked upload endpoint).
HTTP_URL = f"http://{API_URL}"

# How long (seconds) to wait for a response or pulse before giving up.
DEFAULT_TIMEOUT = 5.0

# How long to wait when we expect NO response (e.g. unknown message type).
NO_RESPONSE_TIMEOUT = 1.5