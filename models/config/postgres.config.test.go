package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetTest = postgrespresets.NewDefault(pgdriver.WithDSN(getEnv("POSTGRES_DSN_TEST")))
