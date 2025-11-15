package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/internal/services"
)

func TestTokenRefresh(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.TokenRefreshRequest
		resp *services.Token
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

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				resp: &services.Token{
					AccessToken:  "new-access-token",
					RefreshToken: "refresh_token",
				},
			},

			expectResponse: map[string]any{
				"accessToken":  "new-access-token",
				"refreshToken": "refresh_token",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/InvalidAccessToken",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				err: services.ErrTokenRefreshInvalidAccessToken,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRefreshToken",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				err: services.ErrTokenRefreshInvalidRefreshToken,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MismatchClaims",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				err: services.ErrTokenRefreshMismatchClaims,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MismatchSource",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				err: services.ErrTokenRefreshMismatchSource,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"accessToken": "access-token",
				"refreshToken": "refresh_token"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenRefreshRequest{
					AccessToken:  "access-token",
					RefreshToken: "refresh_token",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockTokenRefreshService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewTokenRefresh(service)
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
