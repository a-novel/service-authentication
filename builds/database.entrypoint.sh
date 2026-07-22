#!/bin/bash
# Wrapper around the postgres image entrypoint, defaulting POSTGRES_DB before it runs.

set -e

POSTGRES_DB=${POSTGRES_DB:-postgres}

exec /usr/local/bin/docker-entrypoint.sh "$@"