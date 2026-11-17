package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestCredentialsGetSelf(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	claims := &session.Claims{UserID: new(credentialsFixture.ID)}
	credentialsSelfJSON := lo.Assign(credentialsFixtureJSON, map[string]any{"locationConsent": false})

	type serviceMock struct {
		resp *core.CredentialsSelf
		err  error
	}

	testCases := []struct {
		name string

		// noClaims sends the request without the claims the auth middleware sets.
		noClaims bool

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: &core.CredentialsSelf{Credentials: *credentialsFixture}},

			expectStatus:   http.StatusOK,
			expectResponse: credentialsSelfJSON,
		},
		{
			name: "Success/PendingEmail",

			serviceMock: &serviceMock{
				resp: &core.CredentialsSelf{
					Credentials: *credentialsFixture,
					PendingEmail: &core.PendingEmail{
						Email:     "user-new@provider.com",
						ExpiresAt: time.Date(2020, time.February, 4, 12, 0, 0, 0, time.UTC),
					},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: lo.Assign(credentialsSelfJSON, map[string]any{
				"pendingEmail": map[string]any{
					"email":     "user-new@provider.com",
					"expiresAt": "2020-02-04T12:00:00Z",
				},
			}),
		},
		{
			name: "Success/LocationConsent",

			serviceMock: &serviceMock{
				resp: &core.CredentialsSelf{Credentials: *credentialsFixture, LocationConsent: true},
			},

			expectStatus:   http.StatusOK,
			expectResponse: lo.Assign(credentialsSelfJSON, map[string]any{"locationConsent": true}),
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
			name: "Error/MissingClaims",

			noClaims: true,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsGetSelfService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.CredentialsGetSelfRequest{ID: credentialsFixture.ID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			ctx := session.SetClaimsContext(t.Context(), claims)
			if testCase.noClaims {
				ctx = t.Context()
			}

			handler := handlers.NewRestCredentialsGetSelf(service, configtest.Logger)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/credentials/self", nil)

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
