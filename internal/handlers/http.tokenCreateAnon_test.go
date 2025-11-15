package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestTokenCreateAnon(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
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

			request: httptest.NewRequest(http.MethodPost, "/", nil),

			serviceMock: &serviceMock{
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
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodPost, "/", nil),

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockTokenCreateAnonService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewTokenCreateAnon(service)
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
