package config

import (
	"github.com/samber/lo"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// PostgresPresetDefault is the server's PostgreSQL connection. It panics when a host has no password.
var PostgresPresetDefault = lo.Must(env.Postgres.Preset())

// migrationsSessionParams bounds how long a migration waits for a lock. After three seconds the
// migration gives up, so the queries queued behind it on its table proceed, and the runner retries it
// later.
var migrationsSessionParams = map[string]any{"lock_timeout": "3s"}

// PostgresPresetMigrations is the migrations job's connection.
var PostgresPresetMigrations = lo.Must(env.Postgres.Preset(pgdriver.WithConnParams(migrationsSessionParams)))
