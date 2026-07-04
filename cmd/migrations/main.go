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

	// Inventory the .up.sql files up-front so the recap can report how many were found.
	// bun's migrator (used by RunMigrationsContext) exposes no count, so walking the
	// embed.FS is the simplest stable approximation.
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

// listUpMigrations returns the bare name of every *.up.sql in f, in the lexical order
// WalkDir yields (the timestamp prefix makes that chronological). It feeds the start-of-run
// inventory log only; bun's migrator decides which migrations actually apply.
func listUpMigrations(f fs.FS) []string {
	var out []string

	_ = fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Aborting the walk only skips inventory logging, not the migration run, so
			// the caller discards this error.
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".up.sql") {
			out = append(out, strings.TrimSuffix(path, ".up.sql"))
		}

		return nil
	})

	return out
}
