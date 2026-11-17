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
	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetMigrations))
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))
}
