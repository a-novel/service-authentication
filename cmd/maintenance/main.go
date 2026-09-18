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

var (
	errMaintenanceMissingEmail         = errors.New("account-role requires --email")
	errMaintenanceMissingRole          = errors.New("account-role requires --role")
	errMaintenanceUnexpectedParameters = errors.New("unexpected maintenance operation parameters")
	errMaintenanceUnknownOperation     = errors.New("unknown maintenance operation")
)

type maintenanceOperation interface {
	run(ctx context.Context) error
}

type maintenanceOperationDefinition struct {
	name        string
	synopsis    string
	description string
	parse       func([]string) (maintenanceOperation, error)
}

type maintenanceCommand struct {
	operations []maintenanceOperationDefinition
}

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
	command := newMaintenanceCommand(accountRoleOperationDefinition())

	operation, err := command.parse(args)
	if err != nil {
		return err
	}

	if operation == nil {
		_, err = fmt.Fprint(os.Stdout, command.usage())
		if err != nil {
			return fmt.Errorf("write help: %w", err)
		}

		return nil
	}

	return operation.run(ctx)
}

func (options *accountRoleOptions) run(ctx context.Context) error {
	cfg := config.AppPresetDefault

	_, err := cfg.Permissions.Priority(options.role)
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

func newMaintenanceCommand(operations ...maintenanceOperationDefinition) *maintenanceCommand {
	return &maintenanceCommand{operations: operations}
}

func newMaintenanceOperationDefinition[T maintenanceOperation](
	name string,
	synopsis string,
	description string,
	bindFlags func(*flag.FlagSet) T,
	validate func(T) error,
) maintenanceOperationDefinition {
	return maintenanceOperationDefinition{
		name:        name,
		synopsis:    synopsis,
		description: description,
		parse: func(args []string) (maintenanceOperation, error) {
			flags := flag.NewFlagSet(name, flag.ContinueOnError)
			flags.SetOutput(os.Stderr)

			operation := bindFlags(flags)

			err := flags.Parse(args)
			if err != nil {
				if errors.Is(err, flag.ErrHelp) {
					return nil, nil
				}

				return nil, fmt.Errorf("parse %s parameters: %w", name, err)
			}

			if flags.NArg() != 0 {
				return nil, fmt.Errorf(
					"%w for %s: %s",
					errMaintenanceUnexpectedParameters,
					name,
					strings.Join(flags.Args(), " "),
				)
			}

			err = validate(operation)
			if err != nil {
				return nil, err
			}

			return operation, nil
		},
	}
}

func accountRoleOperationDefinition() maintenanceOperationDefinition {
	return newMaintenanceOperationDefinition(
		"account-role",
		"--email EMAIL --role ROLE [--lang LANG]",
		"Reconcile an existing account or pending registration to an exact role.",
		func(flags *flag.FlagSet) *accountRoleOptions {
			options := &accountRoleOptions{}
			flags.StringVar(&options.email, "email", "", "account email address")
			flags.StringVar(&options.lang, "lang", config.LangEN, "registration email language")
			flags.StringVar(&options.role, "role", "", "exact authentication role")

			return options
		},
		func(options *accountRoleOptions) error {
			if options.email == "" {
				return errMaintenanceMissingEmail
			}

			if options.role == "" {
				return errMaintenanceMissingRole
			}

			return nil
		},
	)
}

func (command *maintenanceCommand) parse(args []string) (maintenanceOperation, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return nil, nil
	}

	for _, operation := range command.operations {
		if operation.name == args[0] {
			return operation.parse(args[1:])
		}
	}

	return nil, fmt.Errorf("%w %q\n%s", errMaintenanceUnknownOperation, args[0], command.usage())
}

func (command *maintenanceCommand) usage() string {
	var usage strings.Builder

	usage.WriteString("Usage:\n")

	for _, operation := range command.operations {
		usage.WriteString("  maintenance ")
		usage.WriteString(operation.name)
		usage.WriteString(" ")
		usage.WriteString(operation.synopsis)
		usage.WriteString("\n")
	}

	usage.WriteString("\nOperations:\n")

	maxNameLength := 0
	for _, operation := range command.operations {
		maxNameLength = max(maxNameLength, len(operation.name))
	}

	for _, operation := range command.operations {
		usage.WriteString("  ")
		usage.WriteString(operation.name)
		usage.WriteString(strings.Repeat(" ", maxNameLength-len(operation.name)))
		usage.WriteString("  ")
		usage.WriteString(operation.description)
		usage.WriteString("\n")
	}

	return usage.String()
}
