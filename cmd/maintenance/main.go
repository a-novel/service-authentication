// Command maintenance runs trusted, one-shot authentication maintenance operations.
// Operations are selected by subcommand so new routines can be added without changing
// the invocation model. Running the image without an operation only displays help.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

const maintenanceUsage = `Usage:
  maintenance account-role --email EMAIL --role ROLE [--lang LANG]

Operations:
  account-role  Reconcile an existing account or pending registration to an exact role.
`

var (
	errMaintenanceMissingEmail         = errors.New("account-role requires --email")
	errMaintenanceMissingRole          = errors.New("account-role requires --role")
	errMaintenanceUnexpectedParameters = errors.New("unexpected account-role parameters")
	errMaintenanceUnknownOperation     = errors.New("unknown maintenance operation")
)

type accountRoleOptions struct {
	email string
	lang  string
	role  string
}

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("maintenance: ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx, os.Args[1:])
	if err != nil {
		log.Print(err)

		return 1
	}

	return 0
}

func run(ctx context.Context, args []string) error {
	options, err := parseMaintenanceArgs(args)
	if err != nil {
		return err
	}

	if options == nil {
		_, err = fmt.Fprint(os.Stdout, maintenanceUsage)
		if err != nil {
			return fmt.Errorf("write help: %w", err)
		}

		return nil
	}

	cfg := config.AppPresetDefault

	_, err = cfg.Permissions.Priority(options.role)
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
		Email: options.email,
		Lang:  options.lang,
		Role:  options.role,
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
		options.email,
		result.Outcome,
		time.Since(startedAt).Round(time.Millisecond),
	)

	return nil
}

func parseMaintenanceArgs(args []string) (*accountRoleOptions, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return nil, nil
	}

	if args[0] != "account-role" {
		return nil, fmt.Errorf("%w %q\n%s", errMaintenanceUnknownOperation, args[0], maintenanceUsage)
	}

	flags := flag.NewFlagSet("account-role", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	options := &accountRoleOptions{}
	flags.StringVar(&options.email, "email", "", "account email address")
	flags.StringVar(&options.lang, "lang", config.LangEN, "registration email language")
	flags.StringVar(&options.role, "role", "", "exact authentication role")

	err := flags.Parse(args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, nil
		}

		return nil, fmt.Errorf("parse account-role parameters: %w", err)
	}

	if flags.NArg() != 0 {
		return nil, fmt.Errorf(
			"%w: %s",
			errMaintenanceUnexpectedParameters,
			strings.Join(flags.Args(), " "),
		)
	}

	if options.email == "" {
		return nil, errMaintenanceMissingEmail
	}

	if options.role == "" {
		return nil, errMaintenanceMissingRole
	}

	return options, nil
}
