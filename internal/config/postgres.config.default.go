package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

var PostgresPresetDefault = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
