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

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestShortCodeCreateRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validBody := `{"email": "new_user@provider.com", "lang": "fr"}`
	serviceRequest := &core.ShortCodeCreateRegisterRequest{
		Email: "new_user@provider.com", Lang: "fr", Role: authconfig.RoleUser,
	}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{},

			expectStatus: http.StatusAccepted,
		},
		{
			// A registered email answers as a new one does, so a caller cannot probe accounts.
			name: "Success/AlreadyExists",

			serviceMock: &serviceMock{err: core.ErrCredentialsAlreadyExists},

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Error/InvalidRequest",

			serviceMock: &serviceMock{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/MailDeliveryUnavailable",

			serviceMock: &serviceMock{err: errors.Join(errFoo, core.ErrMailDeliveryUnavailable)},

			expectStatus: http.StatusServiceUnavailable,
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

			service := handlersmocks.NewMockRestShortCodeCreateRegisterService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestShortCodeCreateRegister(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPut, "/v2/short-code/register", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
