package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRestTokenRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001"))}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		claims *session.Claims
		body   string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			claims: claims,
			body:   `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/InvalidRefreshToken",

			claims: claims,
			body:   `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: core.ErrTokenRevokeInvalidRefreshToken},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRequest",

			claims: claims,
			body:   `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			claims: claims,
			body:   `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MismatchClaims",

			claims: claims,
			body:   `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: core.ErrTokenRevokeMismatchClaims},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedBody",

			claims: claims,
			body:   `{"refreshToken": `,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MissingClaims",

			body: `{"refreshToken": "refresh-token"}`,

			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestTokenRevokeService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.TokenRevokeRequest{UserID: *claims.UserID, RefreshToken: "refresh-token"}).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestTokenRevoke(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims),
				http.MethodPut, "/v2/session/revoke", strings.NewReader(testCase.body),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
