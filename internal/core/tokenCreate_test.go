package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordArgon2ed, err := lib.GenerateArgon2(passwordRaw, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type issueTokenMock struct {
		resp *servicejsonkeys.ClaimsSignResponse
		err  error
	}

	type issueRefreshTokenMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.TokenCreateRequest

		daoMock               *daoMock
		issueRefreshTokenMock *issueRefreshTokenMock
		issueTokenMock        *issueTokenMock

		expect    *core.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleUser",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleAdmin",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleSuperAdmin",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleSuperAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Error/WrongPassword",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleUser,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/IssueToken",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueRefreshToken",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordArgon2ed,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectCredentials",

			request: &core.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mockDao := coremocks.NewMockTokenCreateDao(t)
			serviceSignClaims := coremocks.NewMockTokenCreateServiceSignClaims(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.issueRefreshTokenMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(
						mock.Anything,
						&servicejsonkeys.ClaimsSignRequest{
							Usage: servicejsonkeys.KeyUsageAuthRefresh,
							Payload: lo.Must(grpcf.MarshalJSONAsAny(core.RefreshTokenClaimsForm{
								UserID: testCase.daoMock.resp.ID,
							})),
						},
					).
					Return(
						&servicejsonkeys.ClaimsSignResponse{
							Token: mockUnsignedRefreshToken,
						},
						testCase.issueRefreshTokenMock.err,
					)
			}

			if testCase.issueTokenMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(
						mock.Anything,
						&servicejsonkeys.ClaimsSignRequest{
							Usage: servicejsonkeys.KeyUsageAuth,
							Payload: lo.Must(grpcf.MarshalJSONAsAny(core.AccessTokenClaims{
								UserID:         &testCase.daoMock.resp.ID,
								Roles:          []string{testCase.daoMock.resp.Role},
								RefreshTokenID: mockUnsignedJTI,
							})),
						},
					).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			service := core.NewTokenCreate(mockDao, serviceSignClaims)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
		})
	}
}
