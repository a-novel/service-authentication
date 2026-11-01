package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestRestAuthorizationCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		accountCaller = "system:serviceaccount:account:web"
		redirectURI   = "https://studio.example.com/cb"
		body          = "response_type=code&client_id=studio"
	)

	claims := &session.Claims{
		UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
		RefreshTokenID: "session-id",
		SessionExp:     1_900_000_000,
	}

	type serviceMock struct {
		caller string
		resp   *core.AuthorizationCode
		err    error
	}

	testCases := []struct {
		name string

		caller      string
		claims      *session.Claims
		contentType string

		serviceMock *serviceMock

		expect       any
		expectStatus int
	}{
		{
			name: "Success",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{
				caller: accountCaller, resp: &core.AuthorizationCode{Code: "code", RedirectURI: redirectURI},
			},

			expect:       &handlers.RestAuthorizationCode{Code: "code", RedirectURI: redirectURI},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/Charset",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/x-www-form-urlencoded; charset=utf-8",

			serviceMock: &serviceMock{
				caller: accountCaller, resp: &core.AuthorizationCode{Code: "code", RedirectURI: redirectURI},
			},

			expect:       &handlers.RestAuthorizationCode{Code: "code", RedirectURI: redirectURI},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/OAuth",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{
				caller: accountCaller,
				err: &core.OAuthError{
					Code: "invalid_request", Description: "transform algorithm not supported", RedirectURI: redirectURI,
				},
			},

			expect: &handlers.RestOAuthError{
				Error: "invalid_request", ErrorDescription: "transform algorithm not supported", RedirectURI: redirectURI,
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/NoCaller",

			claims:      claims,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/SessionEnded",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationCodeCreateSessionEnded},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Internal",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{caller: accountCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			// The parameters travel form-encoded, as on the token endpoint.
			name: "Error/JSONBody",

			caller:      accountCaller,
			claims:      claims,
			contentType: "application/json",

			expect: &handlers.RestOAuthError{
				Error: "invalid_request", ErrorDescription: core.ErrOAuthMalformedRequest.Description,
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MissingClaims",

			caller:      accountCaller,
			contentType: "application/x-www-form-urlencoded",

			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestAuthorizationCodeCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.AuthorizationCodeCreateRequest{
						Caller:     testCase.serviceMock.caller,
						UserID:     *claims.UserID,
						SessionID:  "session-id",
						SessionExp: claims.SessionExp,
						Params:     url.Values{"response_type": {"code"}, "client_id": {"studio"}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationCodeCreate(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims),
				http.MethodPost, "/v2/oauth/authorization-code", strings.NewReader(body),
			)
			request.Header.Set("Content-Type", testCase.contentType)

			res := serveAsCaller(t, handler, request, testCase.caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expect != nil {
				requireNoStore(t, res)
				requireJSONBody(t, res, testCase.expect)
			}

			require.NoError(t, res.Body.Close())

			service.AssertExpectations(t)
		})
	}
}
