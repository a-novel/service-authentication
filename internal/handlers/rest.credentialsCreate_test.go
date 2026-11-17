package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestCredentialsCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"email": "user@provider.com", "password": "Louvre", "shortCode": "abcdef"}`
	serviceRequest := &core.CredentialsCreateRequest{Email: "user@provider.com", Password: "Louvre", ShortCode: "abcdef"}

	type serviceMock struct {
		resp *core.Token
		err  error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: tokenFixture},

			expectStatus:   http.StatusOK,
			expectResponse: tokenFixtureJSON,
		},
		{
			name: "Error/CredentialsAlreadyExists",

			serviceMock: &serviceMock{err: core.ErrCredentialsAlreadyExists},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/ShortCodeInvalid",

			serviceMock: &serviceMock{err: core.ErrShortCodeInvalid},

			expectStatus: http.StatusForbidden,
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
			name: "Error/MalformedBody",

			body: `{"email": `,

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsCreate(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/credentials", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				requireJSONBody(t, res, testCase.expectResponse)
			}

			require.NoError(t, res.Body.Close())
		})
	}
}
