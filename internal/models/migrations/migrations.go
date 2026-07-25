// Package migrations embeds the service's SQL schema migrations so the migration
// runner can apply them straight from the compiled binary.
package migrations

import (
	"embed"
)

// Migrations holds the up and down schema migration files, discovered by their
// timestamped filenames and applied in order by the migration runner.
//
//go:generate env GOLIB_UPDATE_SNAPSHOTS=1 go test -run ^TestMigrationRoundtrip$
//go:embed *.sql
var Migrations embed.FS
