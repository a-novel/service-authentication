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

// authorizationCodePurgeTimeout bounds one run. A run cut short leaves the remaining rows to the next.
const authorizationCodePurgeTimeout = 5 * time.Minute

// authorizationCodePurgeOperation deletes expired authorization codes. A schedule runs it; it takes
// no parameters, so the scheduler can run it without argument overrides.
type authorizationCodePurgeOperation struct{}

func authorizationCodePurgeOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		maintenanceOperationDefinition{
			name:        "authorization-code-purge",
			description: "Delete authorization codes that have expired.",
			spanName:    "job.AuthorizationCodePurge",
			timeout:     authorizationCodePurgeTimeout,
		},
		func(*flag.FlagSet) *authorizationCodePurgeOperation { return &authorizationCodePurgeOperation{} },
		func(*authorizationCodePurgeOperation, config.App) error { return nil },
	)
}

func (operation *authorizationCodePurgeOperation) run(ctx context.Context, _ config.App) error {
	purge := core.NewAuthorizationCodePurge(dao.NewPgAuthorizationCodePurge())

	result, err := purge.Exec(ctx)
	log.Printf("authorization-code-purge purged=%d", result.Purged)

	return err
}
