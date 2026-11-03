package handlers_test

import (
	"errors"
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
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestRefuseEndedSessions(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{
		UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
		RefreshTokenID: "00000000-0000-0000-0000-0000000000a1",
	}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		claims *session.Claims

		serviceMock *serviceMock

		expectStatus    int
		expectChallenge string
	}{
		{
			// A caller whose session lasts reaches the route.
			name: "Success",

			claims: claims,

			serviceMock: &serviceMock{},

			expectStatus: http.StatusTeapot,
		},
		{
			// A signed-out session answers as GET /v2/session answers it.
			name: "Error/Revoked",

			claims: claims,

			serviceMock: &serviceMock{err: core.ErrClaimsGetRevoked},

			expectStatus:    http.StatusUnauthorized,
			expectChallenge: middlewares.BearerChallengeInvalidToken,
		},
		{
			name: "Error/Internal",

			claims: claims,

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestRefuseEndedSessionsService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ClaimsGetRequest{Claims: claims}).
					Return(claims, testCase.serviceMock.err)
			}

			// The route answers 418, so a response tells whether the request reached it.
			route := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
			handler := handlers.NewRestRefuseEndedSessions(service, config.LoggerDev).Middleware(route)

			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims), http.MethodGet, "/v2/sessions", nil,
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectChallenge, res.Header.Get("WWW-Authenticate"))

			service.AssertExpectations(t)
		})
	}
}
