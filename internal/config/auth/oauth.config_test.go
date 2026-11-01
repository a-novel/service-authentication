package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestParseClients(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string

		expect    map[string]authconfig.Client
		expectErr error
	}{
		{
			name: "Success",
			value: `{
				"studio": {"redirectURI": "https://studio.example.com/auth/callback", "caller": "system:serviceaccount:studio:web"},
				"forum": {"redirectURI": "https://forum.example.com/auth/callback", "caller": "system:serviceaccount:forum:web"}
			}`,
			expect: map[string]authconfig.Client{
				"studio": {
					RedirectURI: "https://studio.example.com/auth/callback",
					Caller:      "system:serviceaccount:studio:web",
				},
				"forum": {
					RedirectURI: "https://forum.example.com/auth/callback",
					Caller:      "system:serviceaccount:forum:web",
				},
			},
		},
		{
			name:   "Success/Empty",
			value:  `{}`,
			expect: map[string]authconfig.Client{},
		},
		{
			name:      "Error/Malformed",
			value:     `{"studio":`,
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			name:      "Error/NoRedirectURI",
			value:     `{"studio": {"caller": "system:serviceaccount:studio:web"}}`,
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			name:      "Error/NoCaller",
			value:     `{"studio": {"redirectURI": "https://studio.example.com/auth/callback"}}`,
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			// A code exchange finds its client by the caller, so two clients cannot share one.
			name: "Error/SharedCaller",
			value: `{
				"studio": {"redirectURI": "https://studio.example.com/auth/callback", "caller": "system:serviceaccount:web:web"},
				"forum": {"redirectURI": "https://forum.example.com/auth/callback", "caller": "system:serviceaccount:web:web"}
			}`,
			expectErr: authconfig.ErrClientsConfig,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			clients, err := authconfig.ParseClients(testCase.value)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, clients)
		})
	}
}
