package apiclient_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/apiclient"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestSecurityHandlerService(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type authenticateData struct {
		resp *models.AccessTokenClaims
		err  error
	}

	testCases := []struct {
		name string

		accessToken string

		authenticateData *authenticateData

		granted  models.PermissionsConfig
		required map[codegen.OperationName][]models.Permission

		expect    *models.AccessTokenClaims
		expectErr error
	}{
		{
			name: "Success",

			accessToken: "foobarqux",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{"role:1"},
				},
			},

			granted:  models.PermissionsConfig{},
			required: map[codegen.OperationName][]models.Permission{},

			expect: &models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{"role:1"},
			},
		},
		{
			name: "Success/Anon",

			accessToken: "foobarqux",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{"role:1"},
				},
			},

			granted:  models.PermissionsConfig{},
			required: map[codegen.OperationName][]models.Permission{},

			expect: &models.AccessTokenClaims{
				Roles: []models.Role{"role:1"},
			},
		},
		{
			name: "Success/PreserveRefreshToken",

			accessToken: "foobarqux",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					Roles:          []models.Role{"role:1"},
					RefreshTokenID: lo.ToPtr("refresh-token-id"),
				},
			},

			granted:  models.PermissionsConfig{},
			required: map[codegen.OperationName][]models.Permission{},

			expect: &models.AccessTokenClaims{
				Roles:          []models.Role{"role:1"},
				RefreshTokenID: lo.ToPtr("refresh-token-id"),
			},
		},
		{
			name: "Error",

			accessToken: "foobarqux",

			authenticateData: &authenticateData{
				err: errFoo,
			},

			expectErr: apiclient.ErrCheckSession,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSecurityHandlerService(t)

			if testCase.authenticateData != nil {
				source.EXPECT().
					Authenticate(mock.Anything, testCase.accessToken).
					Return(testCase.authenticateData.resp, testCase.authenticateData.err)
			}

			securityHandler, err := api.NewSecurity(
				testCase.required,
				testCase.granted,
				source,
			)
			require.NoError(t, err)

			handler := &api.API{}

			apiServer, err := codegen.NewServer(handler, securityHandler)
			require.NoError(t, err)

			server := httptest.NewServer(apiServer)
			defer server.Close()

			clientSecurityHandler := apiclient.NewSecurityHandlerService(server.URL)

			claims, err := clientSecurityHandler.Authenticate(t.Context(), testCase.accessToken)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, claims)

			source.AssertExpectations(t)
		})
	}
}
