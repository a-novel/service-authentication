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
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{UserID: new(credentialsFixture.ID)}
	validBody := `{"currentPassword": "current password", "email": "new_user@provider.com", "lang": "fr"}`
	serviceRequest := &core.ShortCodeCreateEmailUpdateRequest{
		CurrentPassword: "current password", Email: "new_user@provider.com", Lang: "fr", ID: credentialsFixture.ID,
	}

	type serviceMock struct {
		err error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string
		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{},

			expectStatus: http.StatusAccepted,
		},
		{
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			// A session alone does not move the account to another address.
			name: "Error/WrongCurrentPassword",

			serviceMock: &serviceMock{err: errors.Join(errFoo, lib.ErrInvalidPassword)},

			expectStatus: http.StatusForbidden,
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
		{
			name: "Error/MissingClaims",

			noClaims: true,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestShortCodeCreateEmailUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestShortCodeCreateEmailUpdate(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPut, "/v2/short-code/update-email", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
