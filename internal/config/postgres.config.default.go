package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

// PostgresPresetDefault is the default Postgres connection preset, built from the
// DSN read from the environment.
var PostgresPresetDefault = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
