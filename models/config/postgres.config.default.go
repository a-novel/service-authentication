package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetDefault = postgrespresets.NewDefault(pgdriver.WithDSN(getEnv("POSTGRES_DSN")))
