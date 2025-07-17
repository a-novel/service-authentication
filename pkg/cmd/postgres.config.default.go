package cmdpkg

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresConfigDefault = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN"))
