package config

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetTest = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN_TEST"))
