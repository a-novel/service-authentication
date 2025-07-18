package services_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	jkmodels "github.com/a-novel/service-json-keys/models"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestConsumeRefreshToken(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type signClaimsData struct {
		resp string
		err  error
	}

	type verifyClaimsData struct {
		resp *models.AccessTokenClaims
		err  error
	}

	type verifyRefreshTokenClaimsData struct {
		resp *models.RefreshTokenClaims
		err  error
	}

	testCases := []struct {
		name string

		request services.ConsumeRefreshTokenRequest

		selectCredentialsData        *selectCredentialsData
		signClaimsData               *signClaimsData
		verifyClaimsData             *verifyClaimsData
		verifyRefreshTokenClaimsData *verifyRefreshTokenClaimsData

		expect    string
		expectErr error
	}{
		{
			name: "Success",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				resp: &models.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: models.CredentialsRoleAdmin,
				},
			},

			signClaimsData: &signClaimsData{
				resp: "access_token",
			},

			expect: "access_token",
		},

		{
			name: "SignError",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				resp: &models.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Role: models.CredentialsRoleAdmin,
				},
			},

			signClaimsData: &signClaimsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectCredentialsError",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				resp: &models.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			selectCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "VerifyRefreshTokenClaimsError",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "VerifyClaimsError",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},

		{
			name: "InvalidRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				err: jws.ErrInvalidSignature,
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "InvalidAccessToken",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				err: jws.ErrInvalidSignature,
			},

			expectErr: models.ErrUnauthorized,
		},

		{
			name: "NoRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken: "access_token",
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "NoAccessToken",

			request: services.ConsumeRefreshTokenRequest{
				RefreshToken: "refresh_token",
			},

			expectErr: models.ErrUnauthorized,
		},
		{
			name: "NotSameUser",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				resp: &models.RefreshTokenClaims{
					Jti:    "refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				},
			},

			expectErr: services.ErrMismatchRefreshClaims,
		},
		{
			name: "NotSameRefreshToken",

			request: services.ConsumeRefreshTokenRequest{
				AccessToken:  "access_token",
				RefreshToken: "refresh_token",
			},

			verifyClaimsData: &verifyClaimsData{
				resp: &models.AccessTokenClaims{
					UserID:         lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Roles:          []models.Role{models.RoleAdmin},
					RefreshTokenID: lo.ToPtr("refresh_token_id"),
				},
			},

			verifyRefreshTokenClaimsData: &verifyRefreshTokenClaimsData{
				resp: &models.RefreshTokenClaims{
					Jti:    "other_refresh_token_id",
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			expectErr: services.ErrTokenIssuedWithDifferentRefreshToken,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockConsumeRefreshTokenSource(t)

			if testCase.verifyClaimsData != nil {
				source.EXPECT().
					VerifyClaims(
						mock.Anything,
						jkmodels.KeyUsageAuth,
						testCase.request.AccessToken,
						&jkpkg.VerifyClaimsOptions{IgnoreExpired: true},
					).
					Return(testCase.verifyClaimsData.resp, testCase.verifyClaimsData.err)
			}

			if testCase.verifyRefreshTokenClaimsData != nil {
				source.EXPECT().
					VerifyRefreshTokenClaims(
						mock.Anything,
						jkmodels.KeyUsageRefresh,
						testCase.request.RefreshToken,
						(*jkpkg.VerifyClaimsOptions)(nil),
					).
					Return(testCase.verifyRefreshTokenClaimsData.resp, testCase.verifyRefreshTokenClaimsData.err)
			}

			if testCase.selectCredentialsData != nil {
				source.EXPECT().
					SelectCredentials(mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000001")).
					Return(testCase.selectCredentialsData.resp, testCase.selectCredentialsData.err)
			}

			if testCase.signClaimsData != nil {
				source.EXPECT().
					SignClaims(
						mock.Anything,
						jkmodels.KeyUsageAuth,
						&models.AccessTokenClaims{
							UserID: testCase.verifyClaimsData.resp.UserID,
							Roles: []models.Role{
								lo.Switch[models.CredentialsRole, models.Role](
									testCase.selectCredentialsData.resp.Role,
								).
									Case(models.CredentialsRoleAdmin, models.RoleAdmin).
									Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
									Default(models.RoleUser),
							},
							RefreshTokenID: &testCase.verifyRefreshTokenClaimsData.resp.Jti,
						},
					).
					Return(testCase.signClaimsData.resp, testCase.signClaimsData.err)
			}

			service := services.NewConsumeRefreshTokenService(source)

			resp, err := service.ConsumeRefreshToken(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
