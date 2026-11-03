package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
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

func TestRestSessionRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	claims := &session.Claims{UserID: lo.ToPtr(userID), RefreshTokenID: "00000000-0000-0000-0000-0000000000a1"}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		claims    *session.Claims
		sessionID string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			claims:    claims,
			sessionID: sessionID.String(),

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/MalformedSessionID",

			claims:    claims,
			sessionID: "session",

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			claims:    claims,
			sessionID: sessionID.String(),

			serviceMock: &serviceMock{err: core.ErrSessionNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			claims:    claims,
			sessionID: sessionID.String(),

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			sessionID: sessionID.String(),

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestSessionRevokeService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.SessionRevokeRequest{UserID: userID, ID: sessionID}).
					Return(testCase.serviceMock.err)
			}

			// The router sets the path parameter, as it would for /v2/sessions/{sessionID}/revoke.
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("sessionID", testCase.sessionID)

			ctx := session.SetClaimsContext(t.Context(), testCase.claims)
			ctx = context.WithValue(ctx, chi.RouteCtxKey, routeContext)

			handler := handlers.NewRestSessionRevoke(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPut, "/v2/sessions/"+testCase.sessionID+"/revoke", nil,
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
