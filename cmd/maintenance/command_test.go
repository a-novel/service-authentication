package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

var errMaintenanceTestDatabase = errors.New("database")

// maintenanceCommandTestOperation records whether it ran.
type maintenanceCommandTestOperation struct {
	ran bool
}

func (operation *maintenanceCommandTestOperation) run(context.Context, config.App) error {
	operation.ran = true

	return nil
}

// maintenanceCommandTestDatabase is a database that fails to connect.
type maintenanceCommandTestDatabase struct{}

func (maintenanceCommandTestDatabase) DB(context.Context) (*bun.DB, error) {
	return nil, errMaintenanceTestDatabase
}

// Close closes nothing: the database never connected.
func (maintenanceCommandTestDatabase) Close() {}

func TestMaintenanceCommand(t *testing.T) {
	t.Parallel()

	expectedOperation := &maintenanceCommandTestOperation{}
	definition := maintenanceOperationDefinition{
		name:        "example",
		synopsis:    "--value VALUE",
		description: "Run the example operation.",
		parse: func([]string, config.App) (maintenanceOperation, error) {
			return expectedOperation, nil
		},
	}
	command := newMaintenanceCommand(config.App{}, definition)

	usage := `Usage:
  maintenance example --value VALUE

Operations:
  example  Run the example operation.
`

	t.Run("Usage", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, usage, command.usage())
	})

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name             string
			args             []string
			expectDefinition string
			expect           maintenanceOperation
			expectErr        error
		}{
			{name: "Success/Empty"},
			{name: "Success/Help", args: []string{"--help"}},
			{
				name:             "Success/Operation",
				args:             []string{"example"},
				expectDefinition: "example",
				expect:           expectedOperation,
			},
			{name: "Error/UnknownOperation", args: []string{"unknown"}, expectErr: errMaintenanceUnknownOperation},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				resultDefinition, result, err := command.parse(testCase.args)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expectDefinition, resultDefinition.name)
				require.Equal(t, testCase.expect, result)
			})
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name         string
			args         []string
			expectOutput string
			expectErr    error
		}{
			{name: "Success/Help", expectOutput: usage},
			{name: "Error/UnknownOperation", args: []string{"unknown"}, expectErr: errMaintenanceUnknownOperation},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				var output strings.Builder

				err := command.run(t.Context(), testCase.args, &output)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expectOutput, output.String())
			})
		}
	})

	// A database that fails to connect fails the run before the operation starts.
	t.Run("RunOnDatabase", func(t *testing.T) {
		t.Parallel()

		operation := &maintenanceCommandTestOperation{}
		command := newMaintenanceCommand(config.App{Postgres: maintenanceCommandTestDatabase{}})

		err := command.runOnDatabase(t.Context(), time.Minute, operation)
		require.ErrorIs(t, err, errMaintenanceTestDatabase)
		require.False(t, operation.ran)
	})
}
