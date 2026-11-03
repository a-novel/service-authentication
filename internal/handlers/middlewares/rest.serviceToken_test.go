package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

const (
	// serviceTokenAudience is the audience the service token tests answer to.
	serviceTokenAudience = "service-authentication"
	// serviceTokenCaller is the one caller the service token tests allow.
	serviceTokenCaller = "system:serviceaccount:agora:studio"
	// serviceTokenStranger is a caller outside the allowed ones.
	serviceTokenStranger = "system:serviceaccount:agora:stranger"
)

func TestRestServiceToken(t *testing.T) {
	t.Parallel()

	issuer := servicetokentest.NewIssuer(t)
	tokenConfig := servicetoken.Config{
		Issuer:   issuer.URL,
		Audience: serviceTokenAudience,
		Callers:  []string{serviceTokenCaller},
	}

	testCases := []struct {
		name string

		tokenConfig    servicetoken.Config
		issuerOptional bool
		token          string

		expectErr    error
		expectStatus int
	}{
		{
			name: "Success",

			tokenConfig: tokenConfig,
			token:       issuer.Sign(t, issuer.Claims(serviceTokenCaller, serviceTokenAudience)),

			expectStatus: http.StatusNoContent,
		},
		{
			// A local or test stack opts out of the issuer, and then takes every request.
			name: "Success/IssuerOptional",

			issuerOptional: true,

			expectStatus: http.StatusNoContent,
		},
		{
			// An issuer keeps the check on, opt-out or not.
			name: "Error/IssuerOptionalWithIssuer",

			tokenConfig:    tokenConfig,
			issuerOptional: true,

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A deployment that forgets the issuer stops at startup instead of taking every request.
			name: "Error/NoIssuer",

			expectErr: middlewares.ErrServiceTokenIssuerRequired,
		},
		{
			name: "Error/UnknownCaller",

			tokenConfig: tokenConfig,
			token:       issuer.Sign(t, issuer.Claims(serviceTokenStranger, serviceTokenAudience)),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Missing",

			tokenConfig: tokenConfig,

			expectStatus: http.StatusUnauthorized,
		},
	}

	// The router answers 204, so a 401 proves the middleware refused the request before the router
	// read its client IP header or counted it against a rate limit.
	router := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/session/", nil)
			if testCase.token != "" {
				request.Header.Set(servicetoken.Header, testCase.token)
			}

			withServiceToken, err := middlewares.RestServiceToken(
				testCase.tokenConfig, testCase.issuerOptional, &http.Client{}, config.LoggerDev, "/v2/ping",
			)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			w := httptest.NewRecorder()
			withServiceToken(router).ServeHTTP(w, request)

			require.Equal(t, testCase.expectStatus, w.Code)
		})
	}
}
