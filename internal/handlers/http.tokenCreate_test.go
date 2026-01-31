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

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.TokenCreateRequest
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
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				resp: &services.Token{
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
			name: "Error/NotFound",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenCreateRequest{
					Email:    "user@provider.com",
					Password: "Louvre",
				},
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			// Returns 401 to prevent email enumeration.
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/PasswordInvalid",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenCreateRequest{
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

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"email": "user@provider.com",
				"password": "Louvre"
			}`)),

			serviceMock: &serviceMock{
				req: &services.TokenCreateRequest{
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

			service := handlersmocks.NewMockTokenCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewTokenCreate(service, config.LoggerDev)
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
