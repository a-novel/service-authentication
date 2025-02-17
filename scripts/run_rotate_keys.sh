#!/bin/bash

KUBE_FILE="pod.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman kube down ${KUBE_FILE}
}
trap int_handler INT

# Setup test containers.
podman play kube ${KUBE_FILE}

export DSN="postgres://test:test@localhost:5001/test?sslmode=disable"
export LOGGER_COLOR=true

go run cmd/rotatekeys/main.go

# Normal execution: containers are shut down.
podman kube down ${KUBE_FILE}
