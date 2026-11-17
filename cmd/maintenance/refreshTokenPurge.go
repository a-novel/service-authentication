package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// refreshTokenPurgeTimeout bounds one run. A run cut short leaves the remaining rows to the next.
const refreshTokenPurgeTimeout = 5 * time.Minute

// refreshTokenPurgeOperation deletes expired refresh token revocations and the sessions that ended
// core.SessionRetention ago. A schedule runs it; it takes no parameters, so the scheduler can run it
// without argument overrides.
type refreshTokenPurgeOperation struct{}

func refreshTokenPurgeOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		maintenanceOperationDefinition{
			name:        "refresh-token-purge",
			description: "Delete refresh token revocations whose token has expired, and sessions that ended 30 days ago.",
			spanName:    "job.RefreshTokenPurge",
			timeout:     refreshTokenPurgeTimeout,
		},
		func(*flag.FlagSet) *refreshTokenPurgeOperation { return &refreshTokenPurgeOperation{} },
		func(*refreshTokenPurgeOperation, config.App) error { return nil },
	)
}

func (operation *refreshTokenPurgeOperation) run(ctx context.Context, _ config.App) error {
	purge := core.NewRefreshTokenRevocationPurge(dao.NewPgRefreshTokenRevocationPurge(), dao.NewPgSessionPurge())

	result, err := purge.Exec(ctx)
	log.Printf("refresh-token-purge purged=%d sessions_purged=%d", result.Purged, result.SessionsPurged)

	return err
}
