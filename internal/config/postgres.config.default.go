package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// PostgresPresetDefault is the default Postgres connection preset, built from the
// DSN read from the environment.
var PostgresPresetDefault = newPostgresPreset()

// newPostgresPreset builds the connection config with its pool bounded.
//
// The limits belong here rather than on the pool the config hands out: that
// handle is cached, so setting them afterwards only takes effect if it happens
// before anything else asks for a connection — an order nothing enforces, and
// missing it applies them to nothing without an error.
func newPostgresPreset() *postgrespresets.Default {
	preset := postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
	preset.MaxOpenConns = env.PostgresMaxOpenConns
	preset.MaxIdleConns = env.PostgresMaxIdleConns

	return preset
}
