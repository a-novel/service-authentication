package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/worker"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/env"
)

var (
	errMaintenanceUnexpectedParameters = errors.New("unexpected maintenance operation parameters")
	errMaintenanceUnknownOperation     = errors.New("unknown maintenance operation")
)

// maintenanceOperation is an operation whose parameters parsed and validated.
type maintenanceOperation interface {
	// run works on the database ctx carries, before the deadline of its definition's timeout.
	run(ctx context.Context, cfg config.App) error
}

// maintenanceOperationDefinition names an operation and parses its parameters.
type maintenanceOperationDefinition struct {
	name        string
	synopsis    string
	description string

	// spanName names the root span of the operation's job, as job.<Operation>.
	spanName string
	// timeout bounds one run, the database connection included.
	timeout time.Duration

	// parse returns nil without an error when args ask for help.
	parse func(args []string, cfg config.App) (maintenanceOperation, error)
}

// maintenanceCommand runs one operation as a job: one root span, one database connection.
type maintenanceCommand struct {
	cfg        config.App
	operations []maintenanceOperationDefinition
}

func newMaintenanceCommand(cfg config.App, operations ...maintenanceOperationDefinition) *maintenanceCommand {
	return &maintenanceCommand{cfg: cfg, operations: operations}
}

// newMaintenanceOperationDefinition completes definition with a parser that binds an operation's
// flags, then validates the operation against the configuration.
func newMaintenanceOperationDefinition[T maintenanceOperation](
	definition maintenanceOperationDefinition,
	bindFlags func(*flag.FlagSet) T,
	validate func(T, config.App) error,
) maintenanceOperationDefinition {
	definition.parse = func(args []string, cfg config.App) (maintenanceOperation, error) {
		return parseOperation(definition.name, bindFlags, validate, args, cfg)
	}

	return definition
}

// parseOperation reads the flags of the named operation from args and validates them. It returns a
// nil operation when args ask for help.
func parseOperation[T maintenanceOperation](
	name string, bindFlags func(*flag.FlagSet) T, validate func(T, config.App) error, args []string, cfg config.App,
) (maintenanceOperation, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	operation := bindFlags(flags)

	err := flags.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return nil, nil
	}

	if err != nil {
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

	err = validate(operation, cfg)
	if err != nil {
		return nil, err
	}

	return operation, nil
}

// run runs the operation args name, or writes the usage to output when args ask for help. A usage
// error returns before telemetry starts, so it records nothing.
func (command *maintenanceCommand) run(ctx context.Context, args []string, output io.Writer) error {
	definition, operation, err := command.parse(args)
	if err != nil {
		return err
	}

	if operation == nil {
		_, err = fmt.Fprint(output, command.usage())
		if err != nil {
			return fmt.Errorf("write help: %w", err)
		}

		return nil
	}

	otel.SetAppName(command.cfg.App.Name + "-maintenance")

	telemetry := env.JobTelemetry.Otel

	err = otel.Init(telemetry)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	defer telemetry.Flush()

	return worker.Run(ctx, definition.spanName, func(ctx context.Context) error {
		return command.runOnDatabase(ctx, definition.timeout, operation)
	})
}

// runOnDatabase connects to the database, then runs operation on it, all within timeout. It runs
// under the job's root span, so the span records a failed connection too.
func (command *maintenanceCommand) runOnDatabase(
	ctx context.Context, timeout time.Duration, operation maintenanceOperation,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ctx, err := postgres.NewContext(ctx, command.cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer command.cfg.Postgres.Close()

	return operation.run(ctx, command.cfg)
}

// parse returns the definition and the parsed operation args name. It returns a nil operation when
// args ask for help.
func (command *maintenanceCommand) parse(
	args []string,
) (maintenanceOperationDefinition, maintenanceOperation, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return maintenanceOperationDefinition{}, nil, nil
	}

	for _, definition := range command.operations {
		if definition.name == args[0] {
			operation, err := definition.parse(args[1:], command.cfg)

			return definition, operation, err
		}
	}

	return maintenanceOperationDefinition{}, nil, fmt.Errorf(
		"%w %q\n%s", errMaintenanceUnknownOperation, args[0], command.usage(),
	)
}

func (command *maintenanceCommand) usage() string {
	var usage strings.Builder

	usage.WriteString("Usage:\n")

	for _, operation := range command.operations {
		fmt.Fprintf(&usage, "  maintenance %s %s\n", operation.name, operation.synopsis)
	}

	usage.WriteString("\nOperations:\n")

	maxNameLength := 0
	for _, operation := range command.operations {
		maxNameLength = max(maxNameLength, len(operation.name))
	}

	for _, operation := range command.operations {
		fmt.Fprintf(&usage, "  %-*s  %s\n", maxNameLength, operation.name, operation.description)
	}

	return usage.String()
}
