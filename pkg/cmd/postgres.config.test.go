package cmdpkg

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var PostgresConfigTest = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN_TEST"))
