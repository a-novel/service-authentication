package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestRestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The account platform describes its visitor in headers. The router reads the IP from the header
	// REST_CLIENT_IP_HEADER names.
	visitorRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{
		"email": "user@provider.com",
		"password": "Louvre"
	}`))
	visitorRequest.Header.Set("X-Client-Ip", "192.0.2.1")
	visitorRequest.Header.Set(handlers.HeaderClientUserAgent, "Firefox")
	visitorRequest.Header.Set(handlers.HeaderClientRegion, "FR")
	visitorRequest.Header.Set(handlers.HeaderClientCity, "Paris")

	type serviceMock struct {
		req  *core.TokenCreateRequest
		resp *core.Token
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &core.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				resp: &core.Token{
					AccessToken:  "token",
					RefreshToken: "refresh",
				},
			},

			expectResponse: map[string]any{
				"accessToken":  "token",
				"refreshToken": "refresh",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/Visitor",

			request: visitorRequest,

			serviceMock: &serviceMock{
				req: &core.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
					Visitor:  core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"},
				},
				resp: &core.Token{AccessToken: "token", RefreshToken: "refresh"},
			},

			expectResponse: map[string]any{
				"accessToken":  "token",
				"refreshToken": "refresh",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NotFound",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &core.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				err: core.ErrCredentialsByEmailNotFound,
			},

			// Returns 401 to prevent email enumeration.
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/PasswordInvalid",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &core.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				err: lib.ErrInvalidPassword,
			},

			// Returns 401 to prevent email enumeration.
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &core.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestTokenCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := middleware.ClientIPFromHeader("X-Client-IP")(handlers.NewRestTokenCreate(service, config.LoggerDev))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
