package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"
	jkModels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
	testutils "github.com/a-novel/service-authentication/pkg/cmd"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type createCredentialsData struct {
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

	type consumeShortCodeData struct {
		err error
	}

	testCases := []struct {
		name string

		request services.RegisterRequest

		createCredentialsData *createCredentialsData
		issueTokenData        *issueTokenData
		issueRefreshTokenData *issueRefreshTokenData
		consumeShortCodeData  *consumeShortCodeData

		expect    *models.Token
		expectErr error
	}{
		{
			name: "Success",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      models.CredentialsRoleUser,
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
			name: "Error/ConsumeShortCode",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/CreateCredentials",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueToken",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      models.CredentialsRoleUser,
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

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      models.CredentialsRoleUser,
				},
			},

			issueRefreshTokenData: &issueRefreshTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, testutils.PostgresConfigTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				source := servicesmocks.NewMockRegisterSource(t)

				if testCase.consumeShortCodeData != nil {
					source.EXPECT().
						ConsumeShortCode(mock.Anything, services.ConsumeShortCodeRequest{
							Usage:  models.ShortCodeUsageRequestRegister,
							Target: testCase.request.Email,
							Code:   testCase.request.ShortCode,
						}).
						Return(nil, testCase.consumeShortCodeData.err)
				}

				if testCase.createCredentialsData != nil {
					source.EXPECT().
						InsertCredentials(mock.Anything, mock.MatchedBy(func(data dao.InsertCredentialsData) bool {
							return assert.Equal(t, testCase.request.Email, data.Email) &&
								assert.NotEqual(t, uuid.Nil, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.NoError(t, lib.CompareScrypt(testCase.request.Password, data.Password))
						})).
						Return(testCase.createCredentialsData.resp, testCase.createCredentialsData.err)
				}

				if testCase.issueRefreshTokenData != nil {
					source.EXPECT().
						SignClaims(mock.Anything, jkModels.KeyUsageRefresh, models.RefreshTokenClaimsInput{
							UserID: testCase.createCredentialsData.resp.ID,
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
								UserID: &testCase.createCredentialsData.resp.ID,
								Roles: []models.Role{
									lo.Switch[models.CredentialsRole, models.Role](
										testCase.createCredentialsData.resp.Role,
									).
										Case(models.CredentialsRoleAdmin, models.RoleAdmin).
										Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
										Default(models.RoleUser),
								},
								RefreshTokenID: lo.ToPtr(mockUnsignedJTI),
							}).
						Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
				}

				service := services.NewRegisterService(source)

				resp, err := service.Register(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				source.AssertExpectations(t)
			})
		})
	}
}
