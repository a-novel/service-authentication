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
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestCredentialsUpdateLocationConsent(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := credentialsFixture.ID
	claims := &session.Claims{UserID: new(userID)}
	// The core receives a refusal as a choice, unlike a body that omits the field.
	validBody := `{"enabled": false}`
	serviceRequest := &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: new(false)}

	type serviceMock struct {
		// req replaces serviceRequest when set.
		req *core.CredentialsUpdateLocationConsentRequest
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

			expectStatus: http.StatusNoContent,
		},
		{
			// The core refuses a body without the choice.
			name: "Error/InvalidRequest",

			body: `{}`,

			serviceMock: &serviceMock{
				req: &core.CredentialsUpdateLocationConsentRequest{UserID: userID},
				err: core.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/NotFound",

			serviceMock: &serviceMock{err: core.ErrCredentialsNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			serviceMock: &serviceMock{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MalformedBody",

			body: `{"enabled": "yes"}`,

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

			service := handlersmocks.NewMockRestCredentialsUpdateLocationConsentService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, cmp.Or(testCase.serviceMock.req, serviceRequest)).
					Return(testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestCredentialsUpdateLocationConsent(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				ctx, http.MethodPatch, "/v2/credentials/location-consent", strings.NewReader(cmp.Or(testCase.body, validBody)),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)
		})
	}
}
