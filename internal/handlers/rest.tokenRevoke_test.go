package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRestTokenRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	claims := &session.Claims{UserID: new(userID)}
	validBody := `{"refreshToken": "refresh-token"}`
	serviceRequest := &core.TokenRevokeRequest{UserID: userID, RefreshToken: "refresh-token"}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string
		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus    int
		expectChallenge string
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/MismatchClaims",

			serviceMock: &serviceMock{err: core.ErrTokenRevokeMismatchClaims},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRequest",

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"refreshToken": `,

			expectStatus: http.StatusBadRequest,
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

			service := handlersmocks.NewMockRestTokenRevokeService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestTokenRevoke(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPut, "/v2/session/revoke", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectChallenge, res.Header.Get("WWW-Authenticate"))
		})
	}
}
