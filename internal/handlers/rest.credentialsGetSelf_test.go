package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestCredentialsGetSelf(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	credentials := core.Credentials{
		ID:        userID,
		Email:     "user@provider.com",
		Role:      authconfig.RoleUser,
		CreatedAt: time.Date(2018, time.February, 2, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2020, time.February, 2, 12, 0, 0, 0, time.UTC),
	}

	credentialsResponse := map[string]any{
		"id":        "00000000-0000-0000-0000-000000000001",
		"email":     "user@provider.com",
		"role":      authconfig.RoleUser,
		"createdAt": "2018-02-02T12:00:00Z",
		"updatedAt": "2020-02-02T12:00:00Z",
	}

	type serviceMock struct {
		resp *core.CredentialsSelf
		err  error
	}

	testCases := []struct {
		name string

		claims *session.Claims

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			claims: &session.Claims{UserID: lo.ToPtr(userID)},

			serviceMock: &serviceMock{
				resp: &core.CredentialsSelf{Credentials: credentials},
			},

			expectResponse: credentialsResponse,
			expectStatus:   http.StatusOK,
		},
		{
			name: "Success/PendingEmail",

			claims: &session.Claims{UserID: lo.ToPtr(userID)},

			serviceMock: &serviceMock{
				resp: &core.CredentialsSelf{
					Credentials: credentials,
					PendingEmail: &core.PendingEmail{
						Email:     "user-new@provider.com",
						ExpiresAt: time.Date(2020, time.February, 4, 12, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: lo.Assign(credentialsResponse, map[string]any{
				"pendingEmail": map[string]any{
					"email":     "user-new@provider.com",
					"expiresAt": "2020-02-04T12:00:00Z",
				},
			}),
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NotFound",

			claims: &session.Claims{UserID: lo.ToPtr(userID)},

			serviceMock: &serviceMock{
				err: core.ErrCredentialsGetNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			claims: &session.Claims{UserID: lo.ToPtr(userID)},

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/MissingClaims",

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsGetSelfService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.CredentialsGetSelfRequest{ID: userID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsGetSelf(service, config.LoggerDev)
			w := httptest.NewRecorder()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			request = request.WithContext(session.SetClaimsContext(request.Context(), testCase.claims))

			handler.ServeHTTP(w, request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}

			service.AssertExpectations(t)
		})
	}
}
