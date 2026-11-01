package auth_test

import (
	"fmt"
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
			// A local stack's client listens on a loopback host, without TLS (RFC 9700 §2.6).
			name:  "Success/Loopback",
			value: registryJSON("studio", "http://localhost:7010/auth/callback"),
			expect: map[string]authconfig.Client{
				"studio": {RedirectURI: "http://localhost:7010/auth/callback", Caller: "system:serviceaccount:studio:web"},
			},
		},
		{
			name:      "Error/HTTP",
			value:     registryJSON("studio", "http://studio.example.com/auth/callback"),
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			// A redirection endpoint is an absolute URI (RFC 6749 §3.1.2).
			name:      "Error/RelativeRedirectURI",
			value:     registryJSON("studio", "/auth/callback"),
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			// A redirection endpoint has no fragment (RFC 6749 §3.1.2).
			name:      "Error/Fragment",
			value:     registryJSON("studio", "https://studio.example.com/auth/callback#"),
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			// A client ID is printable ASCII (RFC 6749 Appendix A.1).
			name:      "Error/ClientID",
			value:     registryJSON("stüdio", "https://studio.example.com/auth/callback"),
			expectErr: authconfig.ErrClientsConfig,
		},
		{
			// A token request finds its client by the caller, so two clients cannot share one.
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

// registryJSON returns a client registry holding clientID alone, with redirectURI.
func registryJSON(clientID, redirectURI string) string {
	return fmt.Sprintf(`{%q: {"redirectURI": %q, "caller": "system:serviceaccount:studio:web"}}`, clientID, redirectURI)
}

func TestOAuth(t *testing.T) {
	t.Parallel()

	const (
		accountCaller = "system:serviceaccount:account:web"
		studioCaller  = "system:serviceaccount:studio:web"
	)

	studio := map[string]authconfig.Client{
		"studio": {RedirectURI: "https://studio.example.com/auth/callback", Caller: studioCaller},
	}

	testCases := []struct {
		name string

		oauth               authconfig.OAuth
		serviceTokenCallers []string

		expectErr error
	}{
		{
			name: "Success",

			oauth:               authconfig.OAuth{AccountPlatformCaller: accountCaller, Clients: studio},
			serviceTokenCallers: []string{accountCaller, studioCaller, "system:serviceaccount:authentication:smoke"},
		},
		{
			// A server that verifies no service token admits every caller.
			name: "Success/NoServiceTokens",

			oauth: authconfig.OAuth{AccountPlatformCaller: accountCaller, Clients: studio},
		},
		{
			// Without clients, the grant is off and needs no caller.
			name: "Success/NoClients",

			oauth:               authconfig.OAuth{AccountPlatformCaller: accountCaller},
			serviceTokenCallers: []string{studioCaller},
		},
		{
			name: "Error/AccountPlatformClient",

			oauth: authconfig.OAuth{
				AccountPlatformCaller: studioCaller, Clients: studio,
			},

			expectErr: authconfig.ErrClientsConfig,
		},
		{
			name: "Error/ClientCallerNotAdmitted",

			oauth:               authconfig.OAuth{AccountPlatformCaller: accountCaller, Clients: studio},
			serviceTokenCallers: []string{accountCaller},

			expectErr: authconfig.ErrClientsConfig,
		},
		{
			name: "Error/AccountPlatformCallerNotAdmitted",

			oauth:               authconfig.OAuth{AccountPlatformCaller: accountCaller, Clients: studio},
			serviceTokenCallers: []string{studioCaller},

			expectErr: authconfig.ErrClientsConfig,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, testCase.oauth.Check(testCase.serviceTokenCallers), testCase.expectErr)
		})
	}
}
