#!/bin/bash

set -e

podman build --format docker \
  -f ./builds/database.Dockerfile \
  -t ghcr.io/a-novel/service-authentication/database:local .
podman build --format docker \
  -f ./builds/rest.Dockerfile \
  -t ghcr.io/a-novel/service-authentication/rest:local .
podman build --format docker \
  -f ./builds/standalone.Dockerfile \
  -t ghcr.io/a-novel/service-authentication/standalone:local .
podman build --format docker \
  -f ./builds/migrations.Dockerfile \
  -t ghcr.io/a-novel/service-authentication/jobs/migrations:local .
