package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"accessToken": "access-token", "refreshToken": "refresh-token"}`
	serviceRequest := &core.TokenRefreshRequest{AccessToken: "access-token", RefreshToken: "refresh-token"}

	type serviceMock struct {
		resp *core.Token
		err  error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: tokenFixture},

			expectStatus:   http.StatusOK,
			expectResponse: tokenFixtureJSON,
		},
		{
			name: "Error/InvalidAccessToken",

			serviceMock: &serviceMock{err: core.ErrTokenRefreshInvalidAccessToken},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRefreshToken",

			serviceMock: &serviceMock{err: core.ErrTokenRefreshInvalidRefreshToken},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MismatchClaims",

			serviceMock: &serviceMock{err: core.ErrTokenRefreshMismatchClaims},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MismatchSource",

			serviceMock: &serviceMock{err: core.ErrTokenRefreshMismatchSource},

			expectStatus: http.StatusForbidden,
		},
		{
			// The account behind a still-valid refresh token is gone, so the caller signs in again.
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusUnauthorized,
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

			body: `{"accessToken": `,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestTokenRefreshService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestTokenRefresh(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPatch, "/v2/session", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

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
