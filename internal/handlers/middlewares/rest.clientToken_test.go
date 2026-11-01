package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestRefuseClientTokens(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name string

		claims *session.Claims

		expectStatus int
	}{
		{
			name: "Success/AccountPlatformToken",

			claims: &session.Claims{UserID: &userID, Roles: []string{"auth:user"}, SessionID: "session-id"},

			expectStatus: http.StatusOK,
		},
		{
			// An optional-auth route without a token puts no claims on the context.
			name: "Success/NoClaims",

			expectStatus: http.StatusOK,
		},
		{
			name: "Error/ClientToken",

			claims: &session.Claims{UserID: &userID, Roles: []string{"auth:user"}, ClientID: "studio"},

			expectStatus: http.StatusForbidden,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := middlewares.RestRefuseClientTokens(config.LoggerDev)(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
			)

			ctx := t.Context()
			if testCase.claims != nil {
				ctx = session.SetClaimsContext(ctx, testCase.claims)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/credentials/self", nil))

			require.Equal(t, testCase.expectStatus, w.Code)
		})
	}
}
