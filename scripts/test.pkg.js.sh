#!/bin/bash

APP_NAME="service-authentication-integration-test"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

pnpm test

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
