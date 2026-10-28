package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// waitlistMaintenanceTimeout bounds a complete scan across the capped temporary list.
const waitlistMaintenanceTimeout = 10 * time.Minute

// waitlistCleanupOperation requires an explicit flag before changing the private sheet.
type waitlistCleanupOperation struct {
	apply bool
}

func waitlistCleanupOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		"waitlist-cleanup", "[--apply]",
		"Preview registered-account entries in the waitlist; --apply removes them.",
		bindWaitlistCleanupFlags,
		func(*waitlistCleanupOperation) error { return nil },
	)
}

func bindWaitlistCleanupFlags(flags *flag.FlagSet) *waitlistCleanupOperation {
	operation := &waitlistCleanupOperation{}
	flags.BoolVar(&operation.apply, "apply", false, "remove rows belonging to registered accounts")

	return operation
}

func (operation *waitlistCleanupOperation) run(ctx context.Context) error {
	cfg := config.AppPresetDefault

	writer, err := dao.NewGoogleWaitlist(cfg.Waitlist, nil)
	if err != nil {
		return err
	}

	if cfg.Waitlist.URL == "" {
		return fmt.Errorf("configure the waitlist before cleanup: %w", core.ErrWaitlistUnavailable)
	}

	otel.SetAppName(cfg.App.Name + "-maintenance")

	err = otel.Init(config.OtelJobPresetDefault)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer config.OtelJobPresetDefault.Flush()

	ctx, cancel := context.WithTimeout(ctx, waitlistMaintenanceTimeout)
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

	cleanup := core.NewWaitlistCleanup(dao.NewPgCredentialsList(), writer)

	result, err := cleanup.Exec(ctx, &core.WaitlistCleanupRequest{Apply: operation.apply})
	log.Printf("waitlist-cleanup apply=%t scanned=%d matched=%d removed=%d",
		operation.apply, result.Scanned, result.Matched, result.Removed)

	return err
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Print("close database: ", err)
	}
}
