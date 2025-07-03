package pkg_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/pkg"
	pkgmocks "github.com/a-novel/service-authentication/pkg/mocks"
)

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	type sourceData struct {
		resp *models.AccessTokenClaims
		err  error
	}

	type requestData struct {
		token string
		roles []string
	}

	testCases := []struct {
		name string

		request *requestData

		sourceData *sourceData

		expect    *models.AccessTokenClaims
		expectErr error
	}{
		{
			name: "Success",

			request: &requestData{
				token: "access-token",
				roles: []string{"foo"},
			},

			sourceData: &sourceData{
				resp: &models.AccessTokenClaims{
					UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:  []models.Role{models.RoleUser},
				},
			},

			expect: &models.AccessTokenClaims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Roles:  []models.Role{models.RoleUser},
			},
		},
		{
			name: "Success/Anonymous",

			request: &requestData{
				token: "access-token",
				roles: []string{"foo"},
			},

			sourceData: &sourceData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{models.RoleAnon},
				},
			},

			expect: &models.AccessTokenClaims{
				Roles: []models.Role{models.RoleAnon},
			},
		},
		{
			name: "Error/InvalidSignature",

			request: &requestData{
				token: "access-token",
			},

			sourceData: &sourceData{
				err: jws.ErrInvalidSignature,
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "Error/BadAccess",

			request: &requestData{
				token: "access-token",
				roles: []string{"bar"},
			},

			sourceData: &sourceData{
				resp: &models.AccessTokenClaims{
					Roles: []models.Role{models.RoleAnon},
				},
			},

			expectErr: models.ErrForbidden,
		},
		{
			name: "Error/NoToken",

			request: &requestData{
				token: "",
			},

			expectErr: models.ErrUnauthorized,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := pkgmocks.NewMockAuthenticateSource(t)

			if testCase.sourceData != nil {
				source.EXPECT().
					VerifyClaims(
						mock.Anything,
						jkModels.KeyUsageAuth,
						testCase.request.token,
						(*jkPkg.VerifyClaimsOptions)(nil),
					).
					Return(testCase.sourceData.resp, testCase.sourceData.err)
			}

			mockToken := pkgmocks.NewMockToken(t)

			if testCase.request != nil {
				mockToken.EXPECT().
					GetToken().
					Return(testCase.request.token)

				if len(testCase.request.roles) > 0 {
					mockToken.EXPECT().
						GetRoles().
						Return(testCase.request.roles)
				}
			}

			auth, err := pkg.NewHandleBearerAuth[string](source, models.PermissionsConfig{
				Roles: map[models.Role]models.RoleConfig{
					models.RoleAnon: {
						Permissions: []models.Permission{"foo"},
					},
					models.RoleUser: {
						Inherits:    []models.Role{models.RoleAnon},
						Permissions: []models.Permission{"bar"},
					},
				},
			})
			require.NoError(t, err)

			ctx, err := auth.HandleBearerAuth(t.Context(), "test-operation", mockToken)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				claims, err := pkg.GetClaimsContext(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, claims)
			} else {
				_, err = pkg.GetClaimsContext(ctx)
				require.Error(t, err)
			}

			source.AssertExpectations(t)
			mockToken.AssertExpectations(t)
		})
	}
}
