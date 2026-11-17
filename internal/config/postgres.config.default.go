package config

import (
	"github.com/samber/lo"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// PostgresPresetDefault is the server's PostgreSQL connection.
var PostgresPresetDefault = lo.Must(env.Postgres.Preset())

// PostgresPresetMigrations is the migrations job's connection. A migration waiting for a lock gives
// up after three seconds, so the queries queued behind it on its table proceed, and the runner tries
// it again later.
var PostgresPresetMigrations = lo.Must(env.Postgres.Preset(
	pgdriver.WithConnParams(map[string]any{"lock_timeout": "3s"}),
))
