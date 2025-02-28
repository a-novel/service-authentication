#!/bin/bash

KUBE_FILE="pod.test.yaml"
PG_VOLUME="c72e2660cc61435ce08b2201b9f0f110dd152fc33d28638c67fdc48a414405a3"
TEST_TOOL_PKG="gotest.tools/gotestsum@latest"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman kube down ${KUBE_FILE}
    podman volume rm "${PG_VOLUME}" -f
}
trap int_handler INT

# Setup test containers.
podman play kube ${KUBE_FILE}

export DSN="postgres://test:test@localhost:5432/test?sslmode=disable"

go run ${TEST_TOOL_PKG} --format pkgname -- -p 1 -cover $(go list ./... | grep -v /mocks | grep -v /codegen)

# Normal execution: containers are shut down.
podman kube down ${KUBE_FILE}
podman volume rm "${PG_VOLUME}" -f
