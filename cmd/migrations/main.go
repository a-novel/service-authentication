// Command migrations applies pending SQL migrations to the authentication database.
// Run this once on first deploy and after each schema change.
package main

import (
	"context"
	"io/fs"
	"log"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("migrations: ")

	start := time.Now()

	// Inventory the .up.sql files up-front so the recap can say how
	// many migrations were discovered (the underlying helper uses
	// uptrace/bun's Migrate, which doesn't expose a count without
	// invasive wrapping — counting the embed.FS is the simplest
	// stable approximation).
	discovered := listUpMigrations(migrations.Migrations)
	log.Printf("discovered %d migration(s) in models/migrations", len(discovered))
	for _, name := range discovered {
		log.Printf("  · %s", name)
	}

	log.Println("connecting to database...")
	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetDefault))

	log.Println("applying pending migrations...")
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))

	log.Printf("done — %d migration(s) examined, completed in %s",
		len(discovered), time.Since(start).Round(time.Millisecond))
}

// listUpMigrations returns the bare names of every *.up.sql in the
// migrations FS, sorted by their natural lexical order (timestamp
// prefix guarantees chronological). Used for the start-of-run
// inventory log; doesn't decide which are actually applied — bun's
// migrator does that against the schema_migrations table.
func listUpMigrations(f fs.FS) []string {
	var out []string
	_ = fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".up.sql") {
			out = append(out, strings.TrimSuffix(path, ".up.sql"))
		}
		return nil
	})
	return out
}
