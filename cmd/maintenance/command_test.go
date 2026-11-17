package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	gootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/a-novel-kit/golib/otel"

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

func (maintenanceCommandTestDatabase) DBSchema(context.Context, string, bool) (*bun.DB, error) {
	return nil, errMaintenanceTestDatabase
}

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
}

//nolint:paralleltest // Replaces the process-wide tracer provider while the job runs.
func TestMaintenanceCommandJob(t *testing.T) {
	errRun := errors.New("run")
	command := newMaintenanceCommand(config.App{Postgres: maintenanceCommandTestDatabase{}})
	unreachable := &maintenanceCommandTestOperation{}

	testCases := []struct {
		name string
		work func(context.Context) error

		expectStatus codes.Code
		expectErr    error
	}{
		{name: "Success", work: func(context.Context) error { return nil }, expectStatus: codes.Unset},
		{
			name:         "Error/Operation",
			work:         func(context.Context) error { return errRun },
			expectStatus: codes.Error,
			expectErr:    errRun,
		},
		{
			// The operation never runs on a database that fails to connect.
			name: "Error/Database",
			work: func(ctx context.Context) error {
				return command.runOnDatabase(ctx, time.Minute, unreachable)
			},
			expectStatus: codes.Error,
			expectErr:    errMaintenanceTestDatabase,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := gootel.GetTracerProvider()

			gootel.SetTracerProvider(provider)
			t.Cleanup(func() {
				gootel.SetTracerProvider(previous)
				require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
			})

			err := runJob(t.Context(), "job.Example", testCase.work)
			require.ErrorIs(t, err, testCase.expectErr)

			spans := recorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, "job.Example", spans[0].Name())
			require.Equal(t, testCase.expectStatus, spans[0].Status().Code)

			if err != nil {
				require.Contains(t, spans[0].Attributes(), otel.ErrorType(err))
			}
		})
	}

	require.False(t, unreachable.ran)
}
