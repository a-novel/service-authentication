package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestAccountRoleOperation(t *testing.T) {
	t.Parallel()

	definition := accountRoleOperationDefinition()

	testCases := []struct {
		name      string
		args      []string
		expect    maintenanceOperation
		expectErr string
	}{
		{
			name: "Success",
			args: []string{
				"--email", "operator@provider.com",
				"--role", authconfig.RoleAdmin,
				"--lang", authconfig.LangFR,
			},
			expect: &accountRoleOperation{
				email: "operator@provider.com",
				role:  authconfig.RoleAdmin,
				lang:  authconfig.LangFR,
			},
		},
		{
			name: "Success/DefaultLanguage",
			args: []string{
				"--email", "operator@provider.com",
				"--role", authconfig.RoleAdmin,
			},
			expect: &accountRoleOperation{
				email: "operator@provider.com",
				role:  authconfig.RoleAdmin,
				lang:  authconfig.LangEN,
			},
		},
		{
			name:      "Error/MissingEmail",
			args:      []string{"--role", authconfig.RoleAdmin},
			expectErr: "requires --email",
		},
		{
			name:      "Error/MissingRole",
			args:      []string{"--email", "operator@provider.com"},
			expectErr: "requires --role",
		},
		{
			name: "Error/UnexpectedParameters",
			args: []string{
				"--email", "operator@provider.com",
				"--role", authconfig.RoleAdmin,
				"extra",
			},
			expectErr: "unexpected maintenance operation parameters",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := definition.parse(testCase.args)
			if testCase.expectErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, testCase.expectErr)
			}

			require.Equal(t, testCase.expect, result)
		})
	}
}
