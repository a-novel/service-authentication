package main

import (
	"context"

	"github.com/samber/lo"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/models/migrations"
)

func main() {
	ctx := lo.Must(postgres.NewContext(context.Background(), config.PostgresPresetDefault))
	lo.Must0(postgres.RunMigrationsContext(ctx, migrations.Migrations))
}
