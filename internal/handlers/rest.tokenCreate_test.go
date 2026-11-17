package handlers_test

import (
	"cmp"
	"errors"
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

	validBody := `{"email": "user@provider.com", "password": "Louvre"}`
	serviceRequest := &core.TokenCreateRequest{
		Email:    "user@provider.com",
		Password: "Louvre",
		Visitor:  core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"},
	}

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
			// An unknown email answers as a wrong password does, so a caller cannot probe accounts.
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/PasswordInvalid",

			serviceMock: &serviceMock{err: lib.ErrInvalidPassword},

			expectStatus: http.StatusUnauthorized,
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

			service := handlersmocks.NewMockRestTokenCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			// The account platform describes its visitor in headers. The router reads the IP from the
			// header REST_CLIENT_IP_HEADER names.
			handler := middleware.ClientIPFromHeader("X-Client-IP")(handlers.NewRestTokenCreate(service, config.LoggerDev))
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/session", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)
			request.Header.Set("X-Client-Ip", "192.0.2.1")
			request.Header.Set(handlers.HeaderClientUserAgent, "Firefox")
			request.Header.Set(handlers.HeaderClientRegion, "FR")
			request.Header.Set(handlers.HeaderClientCity, "Paris")

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
