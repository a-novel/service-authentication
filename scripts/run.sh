#!/bin/bash

set -e

APP_NAME="service-authentication-local"
PODMAN_FILE="$PWD/builds/podman-compose.yaml"

export DEBUG=true

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT

podman compose --podman-build-args='--format docker' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build --pull-always

go run cmd/migrations/main.go
go run cmd/rest/main.go

podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
