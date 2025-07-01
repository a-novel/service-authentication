package services_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestRegister(t *testing.T) { //nolint:paralleltest
	errFoo := errors.New("foo")

	type createCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type issueRefreshTokenData struct {
		token  string
		claims *jwa.Claims
		err    error
	}

	type issueTokenData struct {
		resp string
		err  error
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

			issueRefreshTokenData: &issueRefreshTokenData{
				token: "refresh-token",
				claims: &jwa.Claims{
					ClaimsCommon: jwa.ClaimsCommon{
						Jti: "refresh-token-id",
					},
				},
			},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: &models.Token{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
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

			issueRefreshTokenData: &issueRefreshTokenData{
				token: "refresh-token",
				claims: &jwa.Claims{
					ClaimsCommon: jwa.ClaimsCommon{
						Jti: "refresh-token-id",
					},
				},
			},

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

	for _, testCase := range testCases { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			ctx, err := lib.NewPostgresContext(t.Context(), os.Getenv("DSN"))
			require.NoError(t, err)

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
					IssueRefreshToken(mock.Anything, services.IssueRefreshTokenRequest{
						Claims: &models.AccessTokenClaims{
							UserID: &testCase.createCredentialsData.resp.ID,
							Roles:  []models.Role{models.RoleUser},
						},
					}).
					Return(
						testCase.issueRefreshTokenData.token,
						testCase.issueRefreshTokenData.claims,
						testCase.issueRefreshTokenData.err,
					)
			}

			if testCase.issueTokenData != nil {
				source.EXPECT().
					IssueToken(mock.Anything, services.IssueTokenRequest{
						UserID:         &testCase.createCredentialsData.resp.ID,
						Roles:          []models.Role{models.RoleUser},
						RefreshTokenID: &testCase.issueRefreshTokenData.claims.Jti,
					}).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewRegisterService(source)

			resp, err := service.Register(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
