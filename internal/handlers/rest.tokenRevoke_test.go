package handlers_test

import (
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

func TestRestTokenRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		body string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			body: `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/InvalidRefreshToken",

			body: `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: core.ErrTokenRevokeInvalidRefreshToken},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRequest",

			body: `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			body: `{"refreshToken": "refresh-token"}`,

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"refreshToken": `,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestTokenRevokeService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.TokenRevokeRequest{RefreshToken: "refresh-token"}).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestTokenRevoke(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/session/revoke", strings.NewReader(testCase.body),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
