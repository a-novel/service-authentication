package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

func TestMaintenanceCommand(t *testing.T) {
	t.Parallel()

	command := newMaintenanceCommand(accountRoleOperationDefinition())

	t.Run("Usage", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, `Usage:
  maintenance account-role --email EMAIL --role ROLE [--lang LANG]

Operations:
  account-role  Reconcile an existing account or pending registration to an exact role.
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
			{
				name: "Success/AccountRole",
				args: []string{
					"account-role",
					"--email", "operator@provider.com",
					"--role", config.RoleAdmin,
					"--lang", config.LangFR,
				},
				expect: &accountRoleOptions{
					email: "operator@provider.com",
					role:  config.RoleAdmin,
					lang:  config.LangFR,
				},
			},
			{
				name: "Success/DefaultLanguage",
				args: []string{
					"account-role",
					"--email", "operator@provider.com",
					"--role", config.RoleAdmin,
				},
				expect: &accountRoleOptions{
					email: "operator@provider.com",
					role:  config.RoleAdmin,
					lang:  config.LangEN,
				},
			},
			{
				name:      "Error/UnknownOperation",
				args:      []string{"unknown"},
				expectErr: "unknown maintenance operation",
			},
			{
				name:      "Error/MissingEmail",
				args:      []string{"account-role", "--role", config.RoleAdmin},
				expectErr: "requires --email",
			},
			{
				name:      "Error/MissingRole",
				args:      []string{"account-role", "--email", "operator@provider.com"},
				expectErr: "requires --role",
			},
			{
				name: "Error/UnexpectedParameters",
				args: []string{
					"account-role",
					"--email", "operator@provider.com",
					"--role", config.RoleAdmin,
					"extra",
				},
				expectErr: "unexpected maintenance operation parameters",
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
}
