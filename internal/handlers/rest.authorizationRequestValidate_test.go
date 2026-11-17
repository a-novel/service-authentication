package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
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
		validQuery    = "response_type=code&client_id=studio&code_challenge_method=S256"
	)

	type serviceMock struct {
		resp *core.AuthorizationRequest
		err  error
	}

	testCases := []struct {
		name string

		// query replaces validQuery when set.
		query string
		// noCaller sends the request without a service token, as to a server that verifies none.
		noCaller bool

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: &core.AuthorizationRequest{RedirectURI: redirectURI}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"redirect_uri": redirectURI},
		},
		{
			// The account platform redirects this refusal to the client.
			name: "Error/Redirected",

			serviceMock: &serviceMock{
				err: &core.OAuthError{Code: "invalid_request", Description: "code challenge required", RedirectURI: redirectURI},
			},

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error":             "invalid_request",
				"error_description": "code challenge required",
				"redirect_uri":      redirectURI,
			},
		},
		{
			// The account platform shows this refusal to the person, and redirects nowhere.
			name: "Error/NotRedirected",

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestUnknownClient},

			expectStatus:   http.StatusBadRequest,
			expectResponse: map[string]any{"error": "invalid_request", "error_description": "unknown client"},
		},
		{
			// The core refuses a request with no caller.
			name: "Error/NoCaller",

			noCaller: true,

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedQuery",

			query: "client_id=%zz",

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error": "invalid_request", "error_description": core.ErrOAuthMalformedRequest.Description,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			caller := accountCaller
			if testCase.noCaller {
				caller = ""
			}

			service := handlersmocks.NewMockRestAuthorizationRequestValidateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.AuthorizationRequestValidateRequest{
						Caller: caller,
						Params: url.Values{
							"response_type": {"code"}, "client_id": {"studio"}, "code_challenge_method": {"S256"},
						},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationRequestValidate(service, configtest.Logger)
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, "/v2/oauth/authorization-request?"+cmp.Or(testCase.query, validQuery), nil,
			)

			res := serveAsCaller(t, handler, request, caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
