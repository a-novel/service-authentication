package handlers_test

import (
	"cmp"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestCredentialsList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// A second account makes the response prove that the order holds.
	admin := *credentialsFixture
	admin.ID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	admin.Role = authconfig.RoleAdmin
	adminJSON := lo.Assign(credentialsFixtureJSON, map[string]any{
		"id": "00000000-0000-0000-0000-000000000002", "role": authconfig.RoleAdmin,
	})

	validPath := "/v2/credentials/all?limit=10&offset=2&roles=role1&roles=role2"
	serviceRequest := &core.CredentialsListRequest{Limit: 10, Offset: 2, Roles: []string{"role1", "role2"}}

	type serviceMock struct {
		resp []*core.Credentials
		err  error
	}

	testCases := []struct {
		name string

		// path replaces validPath when set.
		path string

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			serviceMock: &serviceMock{resp: []*core.Credentials{&admin, credentialsFixture}},

			expectStatus:   http.StatusOK,
			expectResponse: []any{adminJSON, credentialsFixtureJSON},
		},
		{
			name: "Success/Empty",

			serviceMock: &serviceMock{resp: []*core.Credentials{}},

			expectStatus:   http.StatusOK,
			expectResponse: []any{},
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
			name: "Error/MalformedQuery",

			path: "/v2/credentials/all?limit=ten",

			expectStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCredentialsListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewRestCredentialsList(service, config.LoggerDev)
			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, cmp.Or(testCase.path, validPath), nil)

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
