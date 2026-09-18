package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
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

func (command *maintenanceCommand) run(ctx context.Context, args []string, output io.Writer) error {
	operation, err := command.parse(args)
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

	return operation.run(ctx)
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
