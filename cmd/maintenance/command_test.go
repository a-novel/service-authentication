package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/sentryf/sentryftest"
)

type maintenanceCommandTestOperation struct {
	err error
}

func (operation *maintenanceCommandTestOperation) run(_ context.Context) error {
	return operation.err
}

func TestMaintenanceCommand(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run")
	expectedOperation := &maintenanceCommandTestOperation{err: errRun}
	definition := maintenanceOperationDefinition{
		name:        "example",
		synopsis:    "--value VALUE",
		description: "Run the example operation.",
		parse: func([]string) (maintenanceOperation, error) {
			return expectedOperation, nil
		},
	}
	command := newMaintenanceCommand(definition)

	t.Run("Usage", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, `Usage:
  maintenance example --value VALUE

Operations:
  example  Run the example operation.
`, command.usage())
	})

	t.Run("Parse", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name      string
			args      []string
			expect    maintenanceOperation
			expectErr string
		}{
			{name: "Success/Empty"},
			{name: "Success/Help", args: []string{"--help"}},
			{name: "Success/Operation", args: []string{"example"}, expect: expectedOperation},
			{
				name:      "Error/UnknownOperation",
				args:      []string{"unknown"},
				expectErr: "unknown maintenance operation",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				result, err := command.parse(testCase.args)
				if testCase.expectErr == "" {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, testCase.expectErr)
				}

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
			{
				name: "Success/Help",
				expectOutput: `Usage:
  maintenance example --value VALUE

Operations:
  example  Run the example operation.
`,
			},
			{name: "Error/Operation", args: []string{"example"}, expectErr: errRun},
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

//nolint:paralleltest // Records through the process-wide Sentry reporter.
func TestMaintenanceCommandSentry(t *testing.T) {
	testCases := []struct {
		name string

		args         []string
		operationErr error

		// A nil fingerprint expects no event.
		expectFingerprint []string
	}{
		{name: "Success", args: []string{"example"}},
		{name: "Success/Help"},
		{
			name: "Error/Operation",

			args:         []string{"example"},
			operationErr: fmt.Errorf("purge: %w", errors.New("run")),

			expectFingerprint: []string{"example", "*errors.errorString"},
		},
		{name: "Error/UnknownOperation", args: []string{"unknown"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transport := sentryftest.Record(t)

			command := newMaintenanceCommand(maintenanceOperationDefinition{
				name: "example",
				parse: func([]string) (maintenanceOperation, error) {
					return &maintenanceCommandTestOperation{err: testCase.operationErr}, nil
				},
			})

			// TestMaintenanceCommand covers the returned error.
			_ = command.run(t.Context(), testCase.args, io.Discard)

			events := transport.Events()
			if testCase.expectFingerprint == nil {
				require.Empty(t, events)

				return
			}

			require.Len(t, events, 1)
			require.Equal(t, testCase.expectFingerprint, events[0].Fingerprint)
		})
	}
}
