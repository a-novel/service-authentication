package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		redirectURI   = "https://studio.example.com/cb"
		query         = "response_type=code&client_id=studio&code_challenge_method=S256"
	)

	type serviceMock struct {
		caller string
		resp   *core.AuthorizationRequest
		err    error
	}

	testCases := []struct {
		name string

		caller   string
		rawQuery string

		serviceMock *serviceMock

		expect       any
		expectStatus int
	}{
		{
			name: "Success",

			caller:   accountCaller,
			rawQuery: query,

			serviceMock: &serviceMock{caller: accountCaller, resp: &core.AuthorizationRequest{RedirectURI: redirectURI}},

			expect:       &handlers.RestAuthorizationRequest{RedirectURI: redirectURI},
			expectStatus: http.StatusOK,
		},
		{
			// The account platform redirects this refusal to the client.
			name: "Error/Redirected",

			caller:   accountCaller,
			rawQuery: query,

			serviceMock: &serviceMock{
				caller: accountCaller,
				err:    &core.OAuthError{Code: "invalid_request", Description: "code challenge required", RedirectURI: redirectURI},
			},

			expect: &handlers.RestOAuthError{
				Error: "invalid_request", ErrorDescription: "code challenge required", RedirectURI: redirectURI,
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			// The account platform shows this refusal to the person, and redirects nowhere.
			name: "Error/NotRedirected",

			caller:   accountCaller,
			rawQuery: query,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationRequestUnknownClient},

			expect:       &handlers.RestOAuthError{Error: "invalid_request", ErrorDescription: "unknown client"},
			expectStatus: http.StatusBadRequest,
		},
		{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/NoCaller",

			rawQuery: query,

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/Internal",

			caller:   accountCaller,
			rawQuery: query,

			serviceMock: &serviceMock{caller: accountCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedQuery",

			caller:   accountCaller,
			rawQuery: "client_id=%zz",

			expect: &handlers.RestOAuthError{
				Error: "invalid_request", ErrorDescription: core.ErrOAuthMalformedRequest.Description,
			},
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
						Caller: testCase.serviceMock.caller,
						Params: url.Values{
							"response_type": {"code"}, "client_id": {"studio"}, "code_challenge_method": {"S256"},
						},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationRequestValidate(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, "/v2/oauth/authorization-request?"+testCase.rawQuery, nil,
			)

			res := serveAsCaller(t, handler, request, testCase.caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expect != nil {
				requireJSONBody(t, res, testCase.expect)
			}

			require.NoError(t, res.Body.Close())

			service.AssertExpectations(t)
		})
	}
}
