#!/bin/bash

KUBE_FILE="pod.test.yaml"
PG_VOLUME="c72e2660cc61435ce08b2201b9f0f110dd152fc33d28638c67fdc48a414405a3"
TEST_TOOL_PKG="gotest.tools/gotestsum@latest"

# First, we set up a temporary directory to receive the coverage (binary)files...
GOCOVERTMPDIR="$(mktemp -d)"
trap 'rm -rf -- "$GOCOVERTMPDIR"' EXIT

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

# Clear old coverage files.
if [ -d "$GOCOVERTMPDIR" ]; then rm -Rf $GOCOVERTMPDIR; fi
mkdir $GOCOVERTMPDIR

# Execute tests.
#
# Go list has 2 exclusive commands, that only work for a specified use case:
# - `go list -m`: list modules (in workspace)
# - `go list ./...`: list sub packages
#
# Since we need a solution to accommodate both cases, I used a workaround;
#  - `go list -m` starts by listing every module (usually just one when not working with workspaces)
#  - `go list ${mod//$(go list .)/.}/...` list every package inside a given sub module
#    - `go list ${package_list}` will print warnings when provided symlinks, which makes the output unusable (until
#       its sanitized). To resolve this issue, we just edit out the prefix for each module (which normally equals the
#       root module name), to turn those modules into relative paths.
#       eg:
#         github.com/org/repo -> .
#         github.com/org/repo/submodule -> ./submodule
go test -p 1 -race -coverprofile=coverage.txt -json ./...

# Normal execution: containers are shut down.
podman kube down ${KUBE_FILE}
podman volume rm "${PG_VOLUME}" -f
