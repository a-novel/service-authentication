package services_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/grpcf"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestTokenCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordScrypted, err := lib.GenerateScrypt(passwordRaw, lib.ScryptParamsDefault)
	require.NoError(t, err)

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	type issueTokenMock struct {
		resp *jkpkg.ClaimsSignResponse
		err  error
	}

	type issueRefreshTokenMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *services.TokenCreateRequest

		repositoryMock        *repositoryMock
		issueRefreshTokenMock *issueRefreshTokenMock
		issueTokenMock        *issueTokenMock

		expect    *services.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access_token",
				},
			},

			expect: &services.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleUser",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     config.RoleUser,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access_token",
				},
			},

			expect: &services.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleAdmin",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     config.RoleAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access_token",
				},
			},

			expect: &services.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Success/RoleSuperAdmin",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     config.RoleSuperAdmin,
				},
			},

			issueRefreshTokenMock: &issueRefreshTokenMock{},

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access_token",
				},
			},

			expect: &services.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Error/WrongPassword",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
					Role:     config.RoleUser,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/IssueToken",

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
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

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: passwordRaw},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
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

			request: &services.TokenCreateRequest{Email: "user@provider.com", Password: "fake-password"},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			repository := servicesmocks.NewMockTokenCreateRepository(t)
			serviceSignClaims := servicesmocks.NewMockTokenCreateServiceSignClaims(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			if testCase.issueRefreshTokenMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, &jkpkg.ClaimsSignRequest{
						Usage: jkpkg.KeyUsageAuthRefresh,
						Payload: lo.Must(grpcf.InterfaceToProtoAny(services.RefreshTokenClaimsForm{
							UserID: testCase.repositoryMock.resp.ID,
						})),
					}).
					Return(
						&jkpkg.ClaimsSignResponse{
							Token: mockUnsignedRefreshToken,
						},
						testCase.issueRefreshTokenMock.err,
					)
			}

			if testCase.issueTokenMock != nil {
				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, jkpkg.KeyUsageAuth,
						services.AccessTokenClaims{
							UserID:         &testCase.repositoryMock.resp.ID,
							Roles:          []string{testCase.repositoryMock.resp.Role},
							RefreshTokenID: mockUnsignedJTI,
						}).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			service := services.NewTokenCreate(repository, serviceSignClaims)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
		})
	}
}
