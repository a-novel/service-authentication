package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestClaimsGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{
		UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
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

		claims *session.Claims

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			claims: claims,

			serviceMock: &serviceMock{resp: claims},

			expectResponse: map[string]any{
				"userID":         "00000000-0000-0000-0000-000000000001",
				"roles":          []any{"user"},
				"refreshTokenID": "refreshToken",
				"sessionID":      "session",
				"sessionExp":     float64(1900000000),
				"clientID":       "studio",
			},
			expectStatus: http.StatusOK,
		},
		{
			// A token without a session claim answers the session its refresh token opened.
			name: "Success/OwnSession",

			claims: &session.Claims{Roles: []string{"user"}, RefreshTokenID: "refreshToken"},

			serviceMock: &serviceMock{resp: &session.Claims{Roles: []string{"user"}, RefreshTokenID: "refreshToken"}},

			expectResponse: map[string]any{
				"roles":          []any{"user"},
				"refreshTokenID": "refreshToken",
				"sessionID":      "refreshToken",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NoClaims",

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/Revoked",

			claims: claims,

			serviceMock: &serviceMock{err: core.ErrClaimsGetRevoked},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Internal",

			claims: claims,

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestClaimsGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ClaimsGetRequest{Claims: testCase.claims}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestClaimsGet(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims), http.MethodGet, "/v2/session", nil,
			)

			handler.ServeHTTP(w, request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}

			service.AssertExpectations(t)
		})
	}
}
