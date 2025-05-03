package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestBearerAuth(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type authenticateData struct {
		resp *models.AccessTokenClaims
		err  error
	}

	testCases := []struct {
		name string

		authenticateData *authenticateData

		granted  models.PermissionsConfig
		required map[codegen.OperationName][]models.Permission

		operationName codegen.OperationName
		auth          codegen.BearerAuth

		expectErr    error
		expectClaims bool
	}{
		{
			name: "Success",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{"role:1"},
				},
			},

			granted: models.PermissionsConfig{
				Roles: map[models.Role]models.RoleConfig{
					"role:1": {
						Permissions: []models.Permission{"perm:1", "perm:3"},
					},
				},
			},
			required: map[codegen.OperationName][]models.Permission{
				"operation:1": {"perm:1"},
				"operation:2": {"perm:2"},
			},

			operationName: "operation:1",
			auth: codegen.BearerAuth{
				Token: "access-token",
			},

			expectErr:    nil,
			expectClaims: true,
		},
		{
			name: "NoRequiredPermissions",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{"role:1"},
				},
			},

			granted: models.PermissionsConfig{
				Roles: map[models.Role]models.RoleConfig{
					"role:1": {
						Permissions: []models.Permission{"perm:1", "perm:3"},
					},
				},
			},
			required: map[codegen.OperationName][]models.Permission{
				"operation:1": {"perm:1"},
				"operation:2": {"perm:2"},
				"operation:3": {},
			},

			operationName: "operation:3",
			auth: codegen.BearerAuth{
				Token: "access-token",
			},

			expectErr:    nil,
			expectClaims: true,
		},
		{
			name: "Error/LackOfPermissions",

			authenticateData: &authenticateData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{"role:1"},
				},
			},

			granted: models.PermissionsConfig{
				Roles: map[models.Role]models.RoleConfig{
					"role:1": {
						Permissions: []models.Permission{"perm:1", "perm:3"},
					},
				},
			},
			required: map[codegen.OperationName][]models.Permission{
				"operation:1": {"perm:1"},
				"operation:2": {"perm:2"},
				"operation:3": {},
			},

			operationName: "operation:2",
			auth: codegen.BearerAuth{
				Token: "access-token",
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "AuthenticateError",

			authenticateData: &authenticateData{
				err: errFoo,
			},

			granted: models.PermissionsConfig{
				Roles: map[models.Role]models.RoleConfig{
					"role:1": {
						Permissions: []models.Permission{"perm:1", "perm:3"},
					},
				},
			},
			required: map[codegen.OperationName][]models.Permission{
				"operation:1": {"perm:1"},
				"operation:2": {"perm:2"},
			},

			operationName: "operation:1",
			auth: codegen.BearerAuth{
				Token: "access-token",
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSecurityHandlerService(t)

			if testCase.authenticateData != nil {
				source.EXPECT().
					Authenticate(t.Context(), testCase.auth.Token).
					Return(testCase.authenticateData.resp, testCase.authenticateData.err)
			}

			security, err := api.NewSecurity(
				testCase.required,
				testCase.granted,
				source,
			)
			require.NoError(t, err)

			ctx, err := security.HandleBearerAuth(t.Context(), testCase.operationName, testCase.auth)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectClaims {
				claims, err := api.GetSecurityClaims(ctx)
				require.NoError(t, err)
				require.NotNil(t, claims)
			} else {
				require.Error(t, err)
			}

			source.AssertExpectations(t)
		})
	}
}
