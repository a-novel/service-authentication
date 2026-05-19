#!/bin/bash
# Sets up environment variables for local development and testing. Each variable uses the
# assign-if-unset pattern (${VAR:=default}), so pre-exported values are preserved and only
# missing ones are filled in. Source this file before running any local service or test command.

export SUPER_ADMIN_EMAIL="${SUPER_ADMIN_EMAIL:="noreply@agorastoryverse.com"}"
export SUPER_ADMIN_PASSWORD="${SUPER_ADMIN_PASSWORD:="admin"}"

# Dummy master key for local use only — never use this value in production or any shared environment.
export SERVICE_JSON_KEYS_APP_MASTER_KEY="${SERVICE_JSON_KEYS_APP_MASTER_KEY:="fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"}"