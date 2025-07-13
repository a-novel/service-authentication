package testutils

import (
	"os"

	postgrespresets "github.com/a-novel/golib/postgres/presets"
)

var TestDBConfig = postgrespresets.NewDefault(os.Getenv("POSTGRES_DSN_TEST"))
