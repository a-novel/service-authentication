package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
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

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	claims := &session.Claims{UserID: lo.ToPtr(userID)}

	type serviceMock struct {
		request *core.CredentialsUpdateLocationConsentRequest
		err     error
	}

	testCases := []struct {
		name string

		claims *session.Claims
		body   string

		serviceMock *serviceMock

		expectStatus int
	}{
		{
			name: "Success",

			claims: claims,
			body:   `{"enabled": false}`,

			serviceMock: &serviceMock{
				request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(false)},
			},

			expectStatus: http.StatusNoContent,
		},
		{
			// The core refuses a body without the choice.
			name: "Error/MissingEnabled",

			claims: claims,
			body:   `{}`,

			serviceMock: &serviceMock{
				request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID},
				err:     core.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/MalformedBody",

			claims: claims,
			body:   `{"enabled": "yes"}`,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			claims: claims,
			body:   `{"enabled": true}`,

			serviceMock: &serviceMock{
				request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(true)},
				err:     core.ErrCredentialsUpdateLocationConsentNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			claims: claims,
			body:   `{"enabled": true}`,

			serviceMock: &serviceMock{
				request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(true)},
				err:     errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			body: `{"enabled": true}`,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsUpdateLocationConsentService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.request).
					Return(testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsUpdateLocationConsent(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				session.SetClaimsContext(t.Context(), testCase.claims),
				http.MethodPatch, "/v2/credentials/location-consent", strings.NewReader(testCase.body),
			)

			handler.ServeHTTP(w, request)

			res := w.Result()
			require.NoError(t, res.Body.Close())
			require.Equal(t, testCase.expectStatus, res.StatusCode)

			service.AssertExpectations(t)
		})
	}
}
