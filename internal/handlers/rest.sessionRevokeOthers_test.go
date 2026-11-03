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
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestSessionRevokeOthers(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	// A token without a session claim opened its own session, so its refresh token names it.
	claims := &session.Claims{UserID: lo.ToPtr(userID), RefreshTokenID: "00000000-0000-0000-0000-0000000000a1"}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		claims *session.Claims

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			claims: claims,

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
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

			service := handlersmocks.NewMockRestSessionRevokeOthersService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.SessionRevokeOthersRequest{UserID: userID, SessionID: claims.RefreshTokenID}).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestSessionRevokeOthers(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims), http.MethodPut, "/v2/sessions/revoke-others", nil,
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
