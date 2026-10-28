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
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
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
	flags.StringVar(&operation.lang, "lang", authconfig.LangEN, "registration email language")
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

	err = otel.Init(config.OtelJobPresetDefault)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer config.OtelJobPresetDefault.Flush()

	if env.GcloudProjectID == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	startedAt := time.Now()

	log.Println("connecting to database")

	ctx, err = postgres.NewContext(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	database, err := cfg.Postgres.DB(ctx)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer closeDatabase(database)

	selectCredentials := dao.NewPgCredentialsSelectByEmail()
	selectShortCode := dao.NewPgShortCodeSelect()
	updateRole := dao.NewPgCredentialsUpdateRole()
	// The job sends its own mail and waits for it, and main reports the failure it returns.
	mailOutbox := core.NewMailOutbox(
		dao.NewPgMailOutboxList(), dao.NewPgMailOutboxClaim(), dao.NewPgShortCodeRenew(),
		dao.NewPgMailOutboxReschedule(), dao.NewPgMailOutboxDelete(),
		cfg.Smtp, env.SmtpMaxConcurrent, nil, cfg.ShortCodesConfig, cfg.SmtpUrlsConfig,
	)
	mail := &deliverNow{outbox: mailOutbox}
	createShortCode := core.NewShortCodeCreate(dao.NewPgShortCodeInsert(), mail, cfg.ShortCodesConfig)
	register := core.NewShortCodeCreateRegister(createShortCode, selectCredentials, cfg.ShortCodesConfig)
	reconcileRole := core.NewCredentialsReconcileRole(selectCredentials, updateRole, selectShortCode, register)

	reconcileRequest := &core.CredentialsReconcileRoleRequest{
		Email: operation.email,
		Lang:  operation.lang,
		Role:  operation.role,
	}

	result, operationErr := reconcileRole.Exec(ctx, reconcileRequest)
	deliveryErr := mail.err

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

// deliverNow sends each mail as soon as it is stored and keeps the failures, so the job waits for its
// own mail. A transient failure stays in the outbox, and the REST API retries it.
type deliverNow struct {
	outbox *core.MailOutbox
	err    error
}

// Enqueue delivers job at once.
func (mail *deliverNow) Enqueue(ctx context.Context, job *core.MailOutboxJob) {
	mail.err = errors.Join(mail.err, mail.outbox.Deliver(ctx, job))
}
