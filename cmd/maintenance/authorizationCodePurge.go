package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// authorizationCodePurgeTimeout bounds one run. A run cut short leaves the remaining rows to the next.
const authorizationCodePurgeTimeout = 5 * time.Minute

// authorizationCodePurgeOperation deletes expired authorization codes. A schedule runs it; it takes
// no parameters, so the scheduler can run it without argument overrides.
type authorizationCodePurgeOperation struct{}

func authorizationCodePurgeOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		"authorization-code-purge", "",
		"Delete authorization codes that have expired.",
		func(*flag.FlagSet) *authorizationCodePurgeOperation { return &authorizationCodePurgeOperation{} },
		func(*authorizationCodePurgeOperation) error { return nil },
	)
}

func (operation *authorizationCodePurgeOperation) run(ctx context.Context) error {
	cfg := config.AppPresetDefault

	otel.SetAppName(cfg.App.Name + "-maintenance")

	err := otel.Init(config.OtelJobPresetDefault)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer config.OtelJobPresetDefault.Flush()

	ctx, cancel := context.WithTimeout(ctx, authorizationCodePurgeTimeout)
	defer cancel()

	ctx, err = postgres.NewContext(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	database, err := cfg.Postgres.DB(ctx)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer closeDatabase(database)

	purge := core.NewAuthorizationCodePurge(dao.NewPgAuthorizationCodePurge())

	result, err := purge.Exec(ctx)
	log.Printf("authorization-code-purge purged=%d", result.Purged)

	return err
}
