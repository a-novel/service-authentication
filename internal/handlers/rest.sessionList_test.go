package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestSessionList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	// A token of a session another token opened names that session.
	claims := &session.Claims{
		UserID: new(userID), RefreshTokenID: "refresh-token-id", SessionID: "00000000-0000-0000-0000-0000000000a1",
	}
	date := time.Date(2026, time.November, 3, 12, 0, 0, 0, time.UTC)

	type serviceMock struct {
		resp []*core.Session
		err  error
	}

	testCases := []struct {
		name string

		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			// A session that recorded no location omits it.
			name: "Success",

			serviceMock: &serviceMock{
				resp: []*core.Session{
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), Current: true, UserAgent: "Firefox",
						IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
						CreatedAt: date, LastActiveAt: date, ExpiresAt: date,
					},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a2"), LoggedOut: true,
						CreatedAt: date, LastActiveAt: date, ExpiresAt: date,
					},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: []any{
				map[string]any{
					"id":           "00000000-0000-0000-0000-0000000000a1",
					"current":      true,
					"loggedOut":    false,
					"userAgent":    "Firefox",
					"ipAddress":    "192.0.2.1",
					"region":       "FR",
					"city":         "Paris",
					"createdAt":    "2026-11-03T12:00:00Z",
					"lastActiveAt": "2026-11-03T12:00:00Z",
					"expiresAt":    "2026-11-03T12:00:00Z",
				},
				map[string]any{
					"id":           "00000000-0000-0000-0000-0000000000a2",
					"current":      false,
					"loggedOut":    true,
					"userAgent":    "",
					"createdAt":    "2026-11-03T12:00:00Z",
					"lastActiveAt": "2026-11-03T12:00:00Z",
					"expiresAt":    "2026-11-03T12:00:00Z",
				},
			},
		},
		{
			name: "Success/Empty",

			serviceMock: &serviceMock{resp: []*core.Session{}},

			expectStatus:   http.StatusOK,
			expectResponse: []any{},
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			noClaims: true,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestSessionListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.SessionListRequest{UserID: userID, SessionID: claims.SessionID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestSessionList(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/sessions", nil)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
