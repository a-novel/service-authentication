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

func TestCredentialsExist(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.CredentialsExistRequest
		resp bool
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
			name: "Exists",

			request: httptest.NewRequest(http.MethodGet, "/?email=user@provider.com", nil),

			serviceMock: &serviceMock{
				req:  &services.CredentialsExistRequest{Email: "user@provider.com"},
				resp: true,
			},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "NotFound",

			request: httptest.NewRequest(http.MethodGet, "/?email=user@provider.com", nil),

			serviceMock: &serviceMock{
				req:  &services.CredentialsExistRequest{Email: "user@provider.com"},
				resp: false,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error",

			request: httptest.NewRequest(http.MethodGet, "/?email=user@provider.com", nil),

			serviceMock: &serviceMock{
				req: &services.CredentialsExistRequest{Email: "user@provider.com"},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockCredentialsExistService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewCredentialsExist(service)
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
