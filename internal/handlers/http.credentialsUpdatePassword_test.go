package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

func TestCredentialsUpdatePassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.CredentialsUpdatePasswordRequest
		resp *services.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request
		claims  *services.AccessTokenClaims

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"password": "Louvre",
				"currentPassword": "abcdef"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.CredentialsUpdatePasswordRequest{
					Password:        "Louvre",
					CurrentPassword: "abcdef",
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				resp: &services.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user@provider.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2018, time.February, 2, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2020, time.February, 2, 12, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000001",
				"email":     "user@provider.com",
				"role":      config.RoleUser,
				"createdAt": "2018-02-02T12:00:00Z",
				"updatedAt": "2020-02-02T12:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/CredentialsNotFound",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"password": "Louvre",
				"currentPassword": "abcdef"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.CredentialsUpdatePasswordRequest{
					Password:        "Louvre",
					CurrentPassword: "abcdef",
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: dao.ErrCredentialsUpdatePasswordNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Pasword",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"password": "Louvre",
				"currentPassword": "abcdef"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.CredentialsUpdatePasswordRequest{
					Password:        "Louvre",
					CurrentPassword: "abcdef",
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: lib.ErrInvalidPassword,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
				"password": "Louvre",
				"currentPassword": "abcdef"
			}`)),
			claims: &services.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.CredentialsUpdatePasswordRequest{
					Password:        "Louvre",
					CurrentPassword: "abcdef",
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockCredentialsUpdatePasswordService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewCredentialsUpdatePassword(service, config.LoggerDev)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx = middlewares.SetClaimsContext(rCtx, testCase.claims)

			handler.ServeHTTP(w, testCase.request.WithContext(rCtx))

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
