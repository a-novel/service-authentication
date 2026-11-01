package handlers_test

import (
	"encoding/json"
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

func TestRestAuthorizationCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		accountCaller = "system:serviceaccount:account:web"
		body          = `{
			"client_id": "studio",
			"redirect_uri": "https://studio.example.com/cb",
			"code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			"code_challenge_method": "S256"
		}`
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

		caller string
		claims *session.Claims
		body   string

		serviceMock *serviceMock

		expect       *handlers.RestAuthorizationCode
		expectStatus int
	}{
		{
			name: "Success",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, resp: &core.AuthorizationCode{Code: "code"}},

			expect:       &handlers.RestAuthorizationCode{Code: "code"},
			expectStatus: http.StatusOK,
		},
		{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/NoCaller",

			claims: claims,
			body:   body,

			serviceMock: &serviceMock{err: core.ErrAuthorizationRequestForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/UnknownClient",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationRequestUnknownClient},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/RedirectURIMismatch",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationRequestRedirectURIMismatch},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/SessionEnded",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrAuthorizationCodeCreateSessionEnded},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/InvalidRequest",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			caller: accountCaller,
			claims: claims,
			body:   body,

			serviceMock: &serviceMock{caller: accountCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			caller: accountCaller,
			claims: claims,
			body:   `{"client_id": `,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MissingClaims",

			caller: accountCaller,
			body:   body,

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
						Caller:              testCase.serviceMock.caller,
						UserID:              *claims.UserID,
						SessionID:           "session-id",
						SessionExp:          claims.SessionExp,
						ClientID:            "studio",
						RedirectURI:         "https://studio.example.com/cb",
						CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
						CodeChallengeMethod: "S256",
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationCodeCreate(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims),
				http.MethodPut, "/v2/authorization-code", strings.NewReader(testCase.body),
			)

			res := serveAsCaller(t, handler, request, testCase.caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expect != nil {
				var body handlers.RestAuthorizationCode

				require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
				require.Equal(t, testCase.expect, &body)
			}

			require.NoError(t, res.Body.Close())

			service.AssertExpectations(t)
		})
	}
}
