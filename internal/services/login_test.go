package services_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jkModels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordScrypted, err := lib.GenerateScrypt(passwordRaw, lib.ScryptParamsDefault)
	require.NoError(t, err)

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type issueTokenData struct {
		resp string
		err  error
	}

	type issueRefreshTokenData struct {
		err error
	}

	testCases := []struct {
		name string

		request services.LoginRequest

		selectCredentialsData *selectCredentialsData
		issueRefreshTokenData *issueRefreshTokenData
		issueTokenData        *issueTokenData

		expect    *models.Token
		expectErr error
	}{
		{
			name: "Success",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleUser,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: &models.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleUser",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleUser,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: &models.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleAdmin",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleAdmin,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: &models.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleSuperAdmin",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleSuperAdmin,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: &models.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Error/WrongPassword",

			request: services.LoginRequest{Email: "user@provider.com", Password: "fake-password"},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleUser,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/IssueToken",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleUser,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{},

			issueTokenData: &issueTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueRefreshToken",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     models.CredentialsRoleUser,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectCredentials",

			request: services.LoginRequest{Email: "user@provider.com", Password: "fake-password"},

			selectCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockLoginSource(t)

			if testCase.selectCredentialsData != nil {
				source.EXPECT().
					SelectCredentialsByEmail(mock.Anything, testCase.request.Email).
					Return(testCase.selectCredentialsData.resp, testCase.selectCredentialsData.err)
			}

			if testCase.issueRefreshTokenData != nil {
				source.EXPECT().
					SignClaims(mock.Anything, jkModels.KeyUsageRefresh, models.RefreshTokenClaimsInput{
						UserID: testCase.selectCredentialsData.resp.ID,
					}).
					Return(
						mockUnsignedRefreshToken,
						testCase.issueRefreshTokenData.err,
					)
			}

			if testCase.issueTokenData != nil {
				source.EXPECT().
					SignClaims(mock.Anything, jkModels.KeyUsageAuth,
						models.AccessTokenClaims{
							UserID: &testCase.selectCredentialsData.resp.ID,
							Roles: []models.Role{
								lo.Switch[models.CredentialsRole, models.Role](
									testCase.selectCredentialsData.resp.Role,
								).
									Case(models.CredentialsRoleAdmin, models.RoleAdmin).
									Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
									Default(models.RoleUser),
							},
							RefreshTokenID: lo.ToPtr(mockUnsignedJTI),
						}).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewLoginService(source)

			resp, err := service.Login(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
