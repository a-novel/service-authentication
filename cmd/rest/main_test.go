package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

const (
	// serviceTokenAudience is the audience the router tests answer to.
	serviceTokenAudience = "service-authentication"
	// serviceTokenCaller is the one caller the router tests allow.
	serviceTokenCaller = "system:serviceaccount:agora:studio"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	issuer := servicetokentest.NewIssuer(t)
	gate := servicetoken.Config{
		Issuer:   issuer.URL,
		Audience: serviceTokenAudience,
		Callers:  []string{serviceTokenCaller},
	}

	// Nothing listens on port 1, so the health route's JSON Keys probe fails at once.
	jsonKeysClient, err := servicejsonkeys.NewClient(
		"localhost:1", grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(jsonKeysClient.Close)

	testCases := []struct {
		name string

		serviceToken servicetoken.Config
		oauth        authconfig.OAuth
		target       string
		token        string

		expectStatus int
		expectErr    error
	}{
		{
			// The liveness probe passes the gate without a token.
			name: "Success/Ping",

			serviceToken: gate,
			target:       "/v2/ping",

			expectStatus: http.StatusOK,
		},
		{
			// An allowed caller reaches the health route. The test hands it no database, so the route
			// answers 503.
			name: "Success/Caller",

			serviceToken: gate,
			target:       "/v2/healthcheck",
			token:        issuer.Sign(t, issuer.Claims(serviceTokenCaller, serviceTokenAudience)),

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			// A local stack opts out, and a request without a token reaches the health route.
			name: "Success/IssuerOptional",

			serviceToken: servicetoken.Config{IssuerOptional: true},
			target:       "/v2/healthcheck",

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Unauthenticated",

			serviceToken: gate,
			target:       "/v2/healthcheck",

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A path outside every route answers 401 too: the gate wraps the whole router, not only
			// the versioned routes.
			name: "Error/UnauthenticatedUnknownPath",

			serviceToken: gate,
			target:       "/nowhere",

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A deployment that forgets the issuer stops at startup rather than admit every caller.
			name: "Error/IssuerRequired",

			expectErr: servicetoken.ErrIssuerRequired,
		},
		{
			// A client that shares the account platform's caller could both issue and exchange codes,
			// so the server stops at startup.
			name: "Error/ClientsConfig",

			serviceToken: gate,
			oauth: authconfig.OAuth{
				AccountPlatformCaller: serviceTokenCaller,
				Clients: map[string]authconfig.Client{
					"studio": {RedirectURI: "https://studio.example.com/auth/callback", Caller: serviceTokenCaller},
				},
			},

			expectErr: authconfig.ErrClientsConfig,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault
			cfg.ServiceToken = testCase.serviceToken
			cfg.OAuth = testCase.oauth
			cfg.Logger = configtest.Logger

			router, err := newRouter(cfg, jsonKeysClient, core.NewMailDelivery(cfg.SMTP, 1))
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.target, nil)
			if testCase.token != "" {
				request.Header.Set(servicetoken.Header, testCase.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			require.Equal(t, testCase.expectStatus, w.Code)
		})
	}
}
