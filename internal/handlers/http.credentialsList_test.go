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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/internal/handlers/mocks"
	"github.com/a-novel/service-authentication/internal/services"
)

func TestCredentialsList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.CredentialsListRequest
		resp []*services.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(http.MethodGet, "/?limit=10&offset=2", nil),

			serviceMock: &serviceMock{
				req: &services.CredentialsListRequest{
					Limit:  10,
					Offset: 2,
				},
				resp: []*services.Credentials{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Email:     "user3@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Email:     "user2@email.com",
						Role:      config.RoleAdmin,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Email:     "user1@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000003",
					"email":     "user3@email.com",
					"role":      config.RoleUser,
					"createdAt": "2021-01-01T00:00:00Z",
					"updatedAt": "2021-01-03T00:00:00Z",
				},
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000002",
					"email":     "user2@email.com",
					"role":      config.RoleAdmin,
					"createdAt": "2021-01-01T00:00:00Z",
					"updatedAt": "2021-01-02T00:00:00Z",
				},
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000001",
					"email":     "user1@email.com",
					"role":      config.RoleUser,
					"createdAt": "2021-01-01T00:00:00Z",
					"updatedAt": "2021-01-01T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/Roles",

			request: httptest.NewRequest(http.MethodGet, "/?limit=10&roles=role1&roles=role2", nil),

			serviceMock: &serviceMock{
				req: &services.CredentialsListRequest{
					Limit: 10,
					Roles: []string{"role1", "role2"},
				},
				resp: []*services.Credentials{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Email:     "user3@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000003",
					"email":     "user3@email.com",
					"role":      config.RoleUser,
					"createdAt": "2021-01-01T00:00:00Z",
					"updatedAt": "2021-01-03T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequest(http.MethodGet, "/?limit=10&offset=2", nil),

			serviceMock: &serviceMock{
				req: &services.CredentialsListRequest{
					Limit:  10,
					Offset: 2,
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockCredentialsListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewCredentialsList(service)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

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
