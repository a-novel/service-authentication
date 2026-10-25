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

// refreshTokenPurgeTimeout bounds one run. A run cut short leaves the remaining rows to the next.
const refreshTokenPurgeTimeout = 5 * time.Minute

// refreshTokenPurgeOperation deletes expired refresh token revocations. A schedule runs it; it takes
// no parameters, so the scheduler can run it without argument overrides.
type refreshTokenPurgeOperation struct{}

func refreshTokenPurgeOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		"refresh-token-purge", "",
		"Delete refresh token revocations whose token has expired.",
		func(*flag.FlagSet) *refreshTokenPurgeOperation { return &refreshTokenPurgeOperation{} },
		func(*refreshTokenPurgeOperation) error { return nil },
	)
}

func (operation *refreshTokenPurgeOperation) run(ctx context.Context) error {
	cfg := config.AppPresetDefault

	otel.SetAppName(cfg.App.Name + "-maintenance")

	err := otel.Init(cfg.Otel)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer cfg.Otel.Flush()

	ctx, cancel := context.WithTimeout(ctx, refreshTokenPurgeTimeout)
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

	purge := core.NewRefreshTokenRevocationPurge(dao.NewPgRefreshTokenRevocationPurge())

	result, err := purge.Exec(ctx)
	log.Printf("refresh-token-purge purged=%d", result.Purged)

	return err
}
