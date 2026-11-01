package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestTokenGrant(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		studioCaller = "system:serviceaccount:studio:web"
		body         = "grant_type=authorization_code&code=code"
	)

	type serviceMock struct {
		caller string
		resp   *core.TokenGrantResult
		err    error
	}

	testCases := []struct {
		name string

		caller      string
		contentType string

		serviceMock *serviceMock

		expect       any
		expectStatus int
	}{
		{
			name: "Success",

			caller:      studioCaller,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{
				caller: studioCaller,
				resp: &core.TokenGrantResult{
					AccessToken: "access-token", ExpiresIn: time.Hour, RefreshToken: "refresh-token",
				},
			},

			expect: &handlers.RestTokenResponse{
				AccessToken: "access-token", TokenType: "Bearer", ExpiresIn: 3600, RefreshToken: "refresh-token",
			},
			expectStatus: http.StatusOK,
		},
		{
			// A refresh issues no refresh token, and the response omits the member.
			name: "Success/Refresh",

			caller:      studioCaller,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{
				caller: studioCaller,
				resp:   &core.TokenGrantResult{AccessToken: "access-token", ExpiresIn: time.Hour},
			},

			expect:       &handlers.RestTokenResponse{AccessToken: "access-token", TokenType: "Bearer", ExpiresIn: 3600},
			expectStatus: http.StatusOK,
		},
		{
			// Every refusal is an error response, 400 (RFC 6749 §5.2).
			name: "Error/InvalidGrant",

			caller:      studioCaller,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{caller: studioCaller, err: core.ErrTokenGrantCodeReused},

			expect: &handlers.RestOAuthError{
				Error: "invalid_grant", ErrorDescription: core.ErrTokenGrantCodeReused.Description,
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			// The core refuses a request with no caller, as on a server that verifies none.
			name: "Error/InvalidClient",

			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{err: core.ErrTokenGrantInvalidClient},

			expect: &handlers.RestOAuthError{
				Error: "invalid_client", ErrorDescription: core.ErrTokenGrantInvalidClient.Description,
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			caller:      studioCaller,
			contentType: "application/x-www-form-urlencoded",

			serviceMock: &serviceMock{caller: studioCaller, err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			// A token request is form-encoded (RFC 6749 §4.1.3).
			name: "Error/JSONBody",

			caller:      studioCaller,
			contentType: "application/json",

			expect: &handlers.RestOAuthError{
				Error: "invalid_request", ErrorDescription: core.ErrOAuthMalformedRequest.Description,
			},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestTokenGrantService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.TokenGrantRequest{
						Caller: testCase.serviceMock.caller,
						Params: url.Values{"grant_type": {"authorization_code"}, "code": {"code"}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestTokenGrant(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPost, "/v2/oauth/token", strings.NewReader(body),
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
