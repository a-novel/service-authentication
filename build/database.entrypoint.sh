#!/bin/bash
set -e

# Default value for POSTGRES_DB if not set
POSTGRES_DB=${POSTGRES_DB:-postgres}

# Execute the original entrypoint script
exec /usr/local/bin/docker-entrypoint.sh "$@"