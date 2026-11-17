// Command migrations applies the pending database schema migrations, then exits. The cmd/rest server
// starts once it completes.
package main

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func main() {
	ctx := context.Background()

	db := lo.Must(config.PostgresPresetMigrations.DB(ctx))
	defer config.PostgresPresetMigrations.Close()

	lo.Must0(postgres.RunMigrations(ctx, db, migrations.Migrations))
}
