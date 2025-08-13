#!/bin/sh
set -e

# Ensure log dir exists and is writable (handles fresh named volumes)
mkdir -p /var/log/uptime
chown -R app:app /var/log/uptime || true

# Drop to non-root and start the API
exec su-exec app /app/api
