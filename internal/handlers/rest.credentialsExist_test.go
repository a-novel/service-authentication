package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestCredentialsExist(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validPath := "/v2/credentials?email=user@provider.com"
	serviceRequest := &core.CredentialsExistRequest{Email: "user@provider.com"}

	type serviceMock struct {
		resp bool
		err  error
	}

	testCases := []struct {
		name string

		// path replaces validPath when set.
		path string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: true},

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Success/NotFound",

			serviceMock: &serviceMock{resp: false},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			// The query decoder refuses a parameter the route does not take.
			name: "Error/MalformedQuery",

			path: "/v2/credentials?mail=user@provider.com",

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsExistService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsExist(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodHead, cmp.Or(testCase.path, validPath), nil)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
