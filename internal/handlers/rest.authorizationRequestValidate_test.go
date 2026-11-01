package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestAuthorizationRequestValidate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		accountCaller = "system:serviceaccount:account:web"
		query         = "/v2/authorization-request?client_id=studio&redirect_uri=https%3A%2F%2Fstudio.example.com%2Fcb"
	)

	type serviceMock struct {
		caller string
		err    error
	}

	testCases := []struct {
		name string

		caller string
		target string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			caller: accountCaller,
			target: query,

			serviceMock: &serviceMock{caller: accountCaller},

			expectStatus: http.StatusNoContent,
		},
		{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/NoCaller",

			target: query,

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/UnknownClient",

			caller: accountCaller,
			target: query,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationRequestUnknownClient},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/RedirectURIMismatch",

			caller: accountCaller,
			target: query,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationRequestRedirectURIMismatch},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			caller: accountCaller,
			target: query,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			caller: accountCaller,
			target: query,

			serviceMock: &serviceMock{caller: accountCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedQuery",

			caller: accountCaller,
			target: "/v2/authorization-request?unknown=1",

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestAuthorizationRequestValidateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.AuthorizationRequestValidateRequest{
						Caller:      testCase.serviceMock.caller,
						ClientID:    "studio",
						RedirectURI: "https://studio.example.com/cb",
					}).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationRequestValidate(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.target, nil)

			res := serveAsCaller(t, handler, request, testCase.caller)
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
