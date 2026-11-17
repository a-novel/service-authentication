package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestClaimsGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{
		UserID:         new(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
		Roles:          []string{"user"},
		RefreshTokenID: "refreshToken",
		SessionID:      "session",
		SessionExp:     1900000000,
		ClientID:       "studio",
	}

	type serviceMock struct {
		resp *session.Claims
		err  error
	}

	testCases := []struct {
		name string

		// claims replaces the shared claims when set; noClaims sends the request without any.
		claims   *session.Claims
		noClaims bool

		serviceMock *serviceMock

		expectStatus    int
		expectChallenge string
		expectResponse  any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: claims},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"userID":         "00000000-0000-0000-0000-000000000001",
				"roles":          []any{"user"},
				"refreshTokenID": "refreshToken",
				"sessionID":      "session",
				"sessionExp":     float64(1900000000),
				"clientID":       "studio",
			},
		},
		{
			// A token without a session claim answers the session its refresh token opened.
			name: "Success/OwnSession",

			claims: &session.Claims{Roles: []string{"user"}, RefreshTokenID: "refreshToken"},

			serviceMock: &serviceMock{resp: &session.Claims{Roles: []string{"user"}, RefreshTokenID: "refreshToken"}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"roles":          []any{"user"},
				"refreshTokenID": "refreshToken",
				"sessionID":      "refreshToken",
			},
		},
		{
			name: "Error/Revoked",

			serviceMock: &serviceMock{err: core.ErrClaimsGetRevoked},

			expectStatus:    http.StatusUnauthorized,
			expectChallenge: middlewares.BearerChallengeInvalidToken,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			noClaims: true,

			expectStatus:    http.StatusUnauthorized,
			expectChallenge: middlewares.BearerChallenge,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			requestClaims := cmp.Or(testCase.claims, claims)
			service := handlersmocks.NewMockRestClaimsGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ClaimsGetRequest{Claims: requestClaims}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), requestClaims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestClaimsGet(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/session", nil)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectChallenge, res.Header.Get("WWW-Authenticate"))

			if testCase.expectResponse != nil {
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
