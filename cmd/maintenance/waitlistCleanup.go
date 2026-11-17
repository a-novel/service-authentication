package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

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
		maintenanceOperationDefinition{
			name:        "waitlist-cleanup",
			synopsis:    "[--apply]",
			description: "Preview registered-account entries in the waitlist; --apply removes them.",
			spanName:    "job.WaitlistCleanup",
			timeout:     waitlistMaintenanceTimeout,
		},
		bindWaitlistCleanupFlags,
		validateWaitlistCleanupOperation,
	)
}

func bindWaitlistCleanupFlags(flags *flag.FlagSet) *waitlistCleanupOperation {
	operation := &waitlistCleanupOperation{}
	flags.BoolVar(&operation.apply, "apply", false, "remove rows belonging to registered accounts")

	return operation
}

// validateWaitlistCleanupOperation requires a configured writer: without one, the cleanup cannot read
// the sheet.
func validateWaitlistCleanupOperation(_ *waitlistCleanupOperation, cfg config.App) error {
	err := cfg.Waitlist.Validate()
	if err != nil {
		return err
	}

	if cfg.Waitlist.URL == "" {
		return fmt.Errorf("configure the waitlist before cleanup: %w", core.ErrWaitlistUnavailable)
	}

	return nil
}

func (operation *waitlistCleanupOperation) run(ctx context.Context, cfg config.App) error {
	writer, err := dao.NewGoogleWaitlist(cfg.Waitlist, nil)
	if err != nil {
		return fmt.Errorf("build waitlist writer: %w", err)
	}

	cleanup := core.NewWaitlistCleanup(dao.NewPgCredentialsList(), writer)

	result, err := cleanup.Exec(ctx, &core.WaitlistCleanupRequest{Apply: operation.apply})
	log.Printf("waitlist-cleanup apply=%t scanned=%d matched=%d removed=%d",
		operation.apply, result.Scanned, result.Matched, result.Removed)

	return err
}
