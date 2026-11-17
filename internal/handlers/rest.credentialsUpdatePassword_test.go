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

func TestRestCredentialsUpdatePassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{UserID: new(credentialsFixture.ID)}
	validBody := `{"password": "Louvre", "currentPassword": "abcdef"}`
	serviceRequest := &core.CredentialsUpdatePasswordRequest{
		Password: "Louvre", CurrentPassword: "abcdef", UserID: credentialsFixture.ID,
	}

	type serviceMock struct {
		resp *core.Credentials
		err  error
	}

	testCases := []struct {
		name string

		// body replaces validBody when set.
		body string
		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: credentialsFixture},

			expectStatus:   http.StatusOK,
			expectResponse: credentialsFixtureJSON,
		},
		{
			name: "Error/CredentialsNotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidPassword",

			serviceMock: &serviceMock{err: lib.ErrInvalidPassword},

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

			body: `{"password": `,

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

			service := handlersmocks.NewMockRestCredentialsUpdatePasswordService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestCredentialsUpdatePassword(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPatch, "/v2/credentials/password", strings.NewReader(cmp.Or(testCase.body, validBody)),
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
