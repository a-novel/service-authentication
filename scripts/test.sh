#!/bin/bash

APP_NAME="service-authentication-test"
PODMAN_FILE="$PWD/builds/podman-compose.test.yaml"

export DEBUG=true

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT

# Setup test containers.
podman compose --podman-build-args='--format docker' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build --pull-always

POSTGRES_DSN=${POSTGRES_DSN_TEST} go run cmd/migrations/main.go

# shellcheck disable=SC2046
PACKAGES="$(go list ./... | grep -v /mocks | grep -v /test)"
go tool gotestsum --format pkgname -- -count=1 -cover $PACKAGES

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
