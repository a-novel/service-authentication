#!/bin/bash

APP_NAME="service-authentication-test"
PODMAN_FILE="$PWD/builds/podman-compose.test.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT EXIT ERR

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# shellcheck disable=SC2046
PACKAGES="$(go list ./... | grep -v /mocks | grep -v /test)"
go tool -modfile=gotestsum.mod gotestsum --format pkgname -- -count=1 -cover $PACKAGES

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
