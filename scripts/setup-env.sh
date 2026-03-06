#!/bin/bash

REST_PORT="${REST_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export REST_PORT
printf "Exposing Rest on port %s\n" "${REST_PORT}"
POSTGRES_PORT="${POSTGRES_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export POSTGRES_PORT
SERVICE_JSON_KEYS_PORT="${SERVICE_JSON_KEYS_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export SERVICE_JSON_KEYS_PORT
MAIL_UI_PORT="${MAIL_UI_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export MAIL_UI_PORT
PLATFORM_AUTH_PORT="${PLATFORM_AUTH_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export PLATFORM_AUTH_PORT

export REST_URL="${REST_URL:="http://localhost:${REST_PORT}"}"
export MAIL_HOST=${MAIL_HOST:="http://localhost:${MAIL_UI_PORT}"}
export PLATFORM_AUTH_URL=""${PLATFORM_AUTH_URL:="http://localhost:${PLATFORM_AUTH_PORT}"}""
export POSTGRES_DSN="${POSTGRES_DSN:="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"}"
