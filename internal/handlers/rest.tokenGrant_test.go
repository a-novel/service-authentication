package handlers_test

import (
	"cmp"
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
		resp *core.TokenGrantResult
		err  error
	}

	testCases := []struct {
		name string

		// contentType replaces the form media type when set.
		contentType string
		// noCaller sends the request without a service token, as to a server that verifies none.
		noCaller bool

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{
				resp: &core.TokenGrantResult{AccessToken: "access-token", ExpiresIn: time.Hour, RefreshToken: "refresh-token"},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"access_token":  "access-token",
				"token_type":    "Bearer",
				"expires_in":    float64(3600),
				"refresh_token": "refresh-token",
			},
		},
		{
			// A refresh issues no refresh token, and the response omits the member.
			name: "Success/Refresh",

			serviceMock: &serviceMock{resp: &core.TokenGrantResult{AccessToken: "access-token", ExpiresIn: time.Hour}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"access_token": "access-token", "token_type": "Bearer", "expires_in": float64(3600),
			},
		},
		{
			// The core refuses a request with no caller. Every refusal is an error response, 400, even
			// invalid_client (RFC 6749 §5.2).
			name: "Error/NoCaller",

			noCaller: true,

			serviceMock: &serviceMock{err: core.ErrTokenGrantInvalidClient},

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error": "invalid_client", "error_description": core.ErrTokenGrantInvalidClient.Description,
			},
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			// A token request is form-encoded (RFC 6749 §4.1.3).
			name: "Error/MalformedBody",

			contentType: "application/json",

			expectStatus: http.StatusBadRequest,
			expectResponse: map[string]any{
				"error": "invalid_request", "error_description": core.ErrOAuthMalformedRequest.Description,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			caller := studioCaller
			if testCase.noCaller {
				caller = ""
			}

			service := handlersmocks.NewMockRestTokenGrantService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.TokenGrantRequest{
						Caller: caller,
						Params: url.Values{"grant_type": {"authorization_code"}, "code": {"code"}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestTokenGrant(service, config.LoggerDev)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v2/oauth/token", strings.NewReader(body))
			request.Header.Set("Content-Type", cmp.Or(testCase.contentType, "application/x-www-form-urlencoded"))

			res := serveAsCaller(t, handler, request, caller)
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				requireNoStore(t, res)
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
