package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

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
		"account-role",
		"--email EMAIL --role ROLE [--lang LANG]",
		"Reconcile an existing account or pending registration to an exact role.",
		bindAccountRoleFlags,
		validateAccountRoleOperation,
	)
}

func bindAccountRoleFlags(flags *flag.FlagSet) *accountRoleOperation {
	operation := &accountRoleOperation{}
	flags.StringVar(&operation.email, "email", "", "account email address")
	flags.StringVar(&operation.lang, "lang", config.LangEN, "registration email language")
	flags.StringVar(&operation.role, "role", "", "exact authentication role")

	return operation
}

func validateAccountRoleOperation(operation *accountRoleOperation) error {
	if operation.email == "" {
		return errMaintenanceMissingEmail
	}

	if operation.role == "" {
		return errMaintenanceMissingRole
	}

	return nil
}

func (operation *accountRoleOperation) run(ctx context.Context) error {
	cfg := config.AppPresetDefault

	_, err := cfg.Permissions.Priority(operation.role)
	if err != nil {
		return fmt.Errorf("invalid --role: %w", err)
	}

	otel.SetAppName(cfg.App.Name + "-maintenance")

	err = otel.Init(cfg.Otel)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	startedAt := time.Now()

	log.Println("connecting to database")

	ctx, err = postgres.NewContext(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	selectCredentials := dao.NewCredentialsSelectByEmail()
	selectShortCode := dao.NewShortCodeSelect()
	updateRole := dao.NewCredentialsUpdateRole()
	createShortCode := core.NewShortCodeCreate(dao.NewShortCodeInsert(), cfg.ShortCodesConfig)
	mailDelivery := core.NewMailDelivery(cfg.Smtp, env.SmtpMaxConcurrent)
	register := core.NewShortCodeCreateRegister(
		createShortCode,
		selectCredentials,
		mailDelivery,
		cfg.ShortCodesConfig,
		cfg.SmtpUrlsConfig,
	)
	reconcileRole := core.NewCredentialsReconcileRole(
		selectCredentials,
		updateRole,
		selectShortCode,
		register,
	)

	result, operationErr := reconcileRole.Exec(ctx, &core.CredentialsReconcileRoleRequest{
		Email: operation.email,
		Lang:  operation.lang,
		Role:  operation.role,
	})

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
