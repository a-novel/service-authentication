package config

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresPresetDefault = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN"))
