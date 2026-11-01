package handlers_test

import (
	"encoding/json"
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

func TestRestAuthorizationCodeExchange(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		studioCaller = "system:serviceaccount:studio:web"
		body         = `{
			"code": "code",
			"code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			"redirect_uri": "https://studio.example.com/cb"
		}`
	)

	type serviceMock struct {
		caller string
		resp   *core.Token
		err    error
	}

	// refusals answer 403 alike, as RFC 6749's invalid_grant does.
	refusals := []error{
		core.ErrAuthorizationCodeExchangeForbiddenCaller,
		core.ErrAuthorizationCodeExchangeUnknownCode,
		core.ErrAuthorizationCodeExchangeReused,
		core.ErrAuthorizationCodeExchangeExpired,
		core.ErrAuthorizationCodeExchangeForeignCode,
		core.ErrAuthorizationCodeExchangeRedirectURIMismatch,
		core.ErrAuthorizationCodeExchangeInvalidVerifier,
		core.ErrAuthorizationCodeExchangeSessionEnded,
	}

	type testCase struct {
		name string

		caller string
		body   string

		serviceMock *serviceMock

		expect       *handlers.Token
		expectStatus int
	}

	testCases := make([]testCase, 0, 5+len(refusals))
	testCases = append(testCases,
		testCase{
			name: "Success",

			caller: studioCaller,
			body:   body,

			serviceMock: &serviceMock{
				caller: studioCaller,
				resp:   &core.Token{AccessToken: "access-token", RefreshToken: "refresh-token"},
			},

			expect:       &handlers.Token{AccessToken: "access-token", RefreshToken: "refresh-token"},
			expectStatus: http.StatusOK,
		},
		testCase{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/NoCaller",

			body: body,

			serviceMock: &serviceMock{err: core.ErrAuthorizationCodeExchangeForbiddenCaller},

			expectStatus: http.StatusForbidden,
		},
		testCase{
			name: "Error/InvalidRequest",

			caller: studioCaller,
			body:   body,

			serviceMock: &serviceMock{caller: studioCaller, err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		testCase{
			name: "Error/Internal",

			caller: studioCaller,
			body:   body,

			serviceMock: &serviceMock{caller: studioCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		testCase{
			name: "Error/MalformedBody",

			caller: studioCaller,
			body:   `{"code": `,

			expectStatus: http.StatusBadRequest,
		},
	)

	for _, refusal := range refusals {
		testCases = append(testCases, testCase{
			name: "Error/Refused/" + strings.ReplaceAll(refusal.Error(), " ", "_"),

			caller: studioCaller,
			body:   body,

			serviceMock: &serviceMock{caller: studioCaller, err: refusal},

			expectStatus: http.StatusForbidden,
		})
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestAuthorizationCodeExchangeService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.AuthorizationCodeExchangeRequest{
						Caller:       testCase.serviceMock.caller,
						Code:         "code",
						CodeVerifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
						RedirectURI:  "https://studio.example.com/cb",
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestAuthorizationCodeExchange(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/session/authorization-code", strings.NewReader(testCase.body),
			)

			res := serveAsCaller(t, handler, request, testCase.caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expect != nil {
				var token handlers.Token

				require.NoError(t, json.NewDecoder(res.Body).Decode(&token))
				require.Equal(t, testCase.expect, &token)
			}

			require.NoError(t, res.Body.Close())

			service.AssertExpectations(t)
		})
	}
}
