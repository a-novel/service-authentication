package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestRestAuthorizationCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		accountCaller = "system:serviceaccount:account:web"
		redirectURI   = "https://studio.example.com/cb"
		body          = "response_type=code&client_id=studio"
	)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	claims := &session.Claims{
		UserID: new(userID), RefreshTokenID: "refresh-token-id", SessionID: "session-id", SessionExp: 1_900_000_000,
	}

	type serviceMock struct {
		resp *core.AuthorizationCode
		err  error
	}

	testCases := []struct {
		name string

		// contentType replaces the form media type when set.
		contentType string
		// noCaller sends the request without a service token, as to a server that verifies none.
		noCaller bool
		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus    int
		expectChallenge string
		expectResponse  any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: &core.AuthorizationCode{Code: "code", RedirectURI: redirectURI}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"code": "code", "redirect_uri": redirectURI},
		},
		{
			name: "Success/Charset",

			contentType: "application/x-www-form-urlencoded; charset=utf-8",

			serviceMock: &serviceMock{resp: &core.AuthorizationCode{Code: "code", RedirectURI: redirectURI}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"code": "code", "redirect_uri": redirectURI},
		},
		{
			name: "Error/OAuth",

			serviceMock: &serviceMock{
				err: &core.OAuthError{
					Code: "invalid_request", Description: "transform algorithm not supported", RedirectURI: redirectURI,
				},
			},

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error":             "invalid_request",
				"error_description": "transform algorithm not supported",
				"redirect_uri":      redirectURI,
			},
		},
		{
			// The core refuses a request with no caller.
			name: "Error/NoCaller",

			noCaller: true,

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/SessionEnded",

			serviceMock: &serviceMock{err: core.ErrAuthorizationCodeCreateSessionEnded},

			expectStatus:    http.StatusUnauthorized,
			expectChallenge: middlewares.BearerChallengeInvalidToken,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			// The parameters travel form-encoded, as on the token endpoint.
			name: "Error/MalformedBody",

			contentType: "application/json",

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error": "invalid_request", "error_description": core.ErrOAuthMalformedRequest.Description,
			},
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

			caller := accountCaller
			if testCase.noCaller {
				caller = ""
			}

			service := handlersmocks.NewMockRestAuthorizationCodeCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.AuthorizationCodeCreateRequest{
						Caller:         caller,
						UserID:         userID,
						RefreshTokenID: "refresh-token-id",
						SessionID:      "session-id",
						SessionExp:     claims.SessionExp,
						Params:         url.Values{"response_type": {"code"}, "client_id": {"studio"}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestAuthorizationCodeCreate(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPost, "/v2/oauth/authorization-code", strings.NewReader(body),
			)
			request.Header.Set("Content-Type", cmp.Or(testCase.contentType, "application/x-www-form-urlencoded"))

			res := serveAsCaller(t, handler, request, caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)
			require.Equal(t, testCase.expectChallenge, res.Header.Get("WWW-Authenticate"))

			if testCase.expectResponse != nil {
				requireNoStore(t, res)
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
