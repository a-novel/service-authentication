package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// accountRoleTimeout bounds one run, the registration email included.
const accountRoleTimeout = 5 * time.Minute

var (
	errMaintenanceMissingEmail = errors.New("account-role requires --email")
	errMaintenanceMissingRole  = errors.New("account-role requires --role")
)

type accountRoleOperation struct {
	email string
	lang  string
	role  string
}

func accountRoleOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		maintenanceOperationDefinition{
			name:        "account-role",
			synopsis:    "--email EMAIL --role ROLE [--lang LANG]",
			description: "Reconcile an existing account or pending registration to an exact role.",
			spanName:    "job.AccountRole",
			timeout:     accountRoleTimeout,
		},
		bindAccountRoleFlags,
		validateAccountRoleOperation,
	)
}

func bindAccountRoleFlags(flags *flag.FlagSet) *accountRoleOperation {
	operation := &accountRoleOperation{}
	flags.StringVar(&operation.email, "email", "", "account email address")
	flags.StringVar(&operation.lang, "lang", authconfig.LangEN, "registration email language")
	flags.StringVar(&operation.role, "role", "", "exact authentication role")

	return operation
}

func validateAccountRoleOperation(operation *accountRoleOperation, cfg config.App) error {
	if operation.email == "" {
		return errMaintenanceMissingEmail
	}

	if operation.role == "" {
		return errMaintenanceMissingRole
	}

	_, err := cfg.Permissions.Priority(operation.role)
	if err != nil {
		return fmt.Errorf("invalid --role: %w", err)
	}

	return nil
}

func (operation *accountRoleOperation) run(ctx context.Context, cfg config.App) error {
	startedAt := time.Now()

	selectCredentials := dao.NewPgCredentialsSelectByEmail()
	selectShortCode := dao.NewPgShortCodeSelect()
	updateRole := dao.NewPgCredentialsUpdateRole()
	createShortCode := core.NewShortCodeCreate(dao.NewPgShortCodeInsert(), cfg.ShortCodes)
	mailDelivery := core.NewMailDelivery(cfg.SMTP, env.SMTPMaxConcurrent)
	register := core.NewShortCodeCreateRegister(
		createShortCode, dao.NewPgCredentialsExist(), mailDelivery, cfg.ShortCodes, cfg.SMTPURLs,
	)
	reconcileRole := core.NewCredentialsReconcileRole(selectCredentials, updateRole, selectShortCode, register)

	reconcileRequest := &core.CredentialsReconcileRoleRequest{
		Email: operation.email,
		Lang:  operation.lang,
		Role:  operation.role,
	}

	result, operationErr := reconcileRole.Exec(ctx, reconcileRequest)

	deliveryErr := mailDelivery.Wait(ctx)

	if operationErr != nil {
		return fmt.Errorf("reconcile account role: %w", errors.Join(operationErr, deliveryErr))
	}

	if deliveryErr != nil {
		return fmt.Errorf("reconcile account role: %w", deliveryErr)
	}

	log.Printf(
		"account-role completed for %s with outcome %s in %s",
		operation.email,
		result.Outcome,
		time.Since(startedAt).Round(time.Millisecond),
	)

	return nil
}
