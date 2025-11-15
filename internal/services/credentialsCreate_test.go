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

	"github.com/a-novel/golib/grpcf"
	"github.com/a-novel/golib/postgres"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestCredentialsCreateRequest(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	type issueTokenMock struct {
		resp *jkpkg.ClaimsSignResponse
		err  error
	}

	type serviceSignClaimsMock struct {
		err error
	}

	type serviceShortCodeConsumeMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *services.CredentialsCreateRequest

		repositoryMock              *repositoryMock
		issueTokenMock              *issueTokenMock
		serviceSignClaimsMock       *serviceSignClaimsMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock

		expect    *services.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      config.RoleUser,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{},

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &services.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
		},
		{
			name: "Error/ConsumeShortCode",

			request: &services.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/CreateCredentials",

			request: &services.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueToken",

			request: &services.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      config.RoleUser,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{},

			issueTokenMock: &issueTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueRefreshToken",

			request: &services.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      config.RoleUser,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				repository := servicesmocks.NewMockCredentialsCreateRepository(t)
				serviceShortCodeConsume := servicesmocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
				serviceSignClaims := servicesmocks.NewMockCredentialsCreateServiceSignClaims(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &services.ShortCodeConsumeRequest{
							Usage:  services.ShortCodeUsageRegister,
							Target: testCase.request.Email,
							Code:   testCase.request.ShortCode,
						}).
						Return(nil, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.repositoryMock != nil {
					repository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
							return assert.Equal(t, testCase.request.Email, data.Email) &&
								assert.NotEqual(t, uuid.Nil, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
								assert.NoError(t, lib.CompareScrypt(testCase.request.Password, data.Password)) &&
								assert.Equal(t, config.RoleUser, data.Role)
						})).
						Return(
							testCase.repositoryMock.resp,
							testCase.repositoryMock.err,
						)
				}

				if testCase.serviceSignClaimsMock != nil {
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
							testCase.serviceSignClaimsMock.err,
						)
				}

				if testCase.issueTokenMock != nil {
					serviceSignClaims.EXPECT().
						ClaimsSign(mock.Anything, &jkpkg.ClaimsSignRequest{
							Usage: jkpkg.KeyUsageAuth,
							Payload: lo.Must(grpcf.InterfaceToProtoAny(services.AccessTokenClaims{
								UserID:         &testCase.repositoryMock.resp.ID,
								Roles:          []string{testCase.repositoryMock.resp.Role},
								RefreshTokenID: mockUnsignedJTI,
							})),
						}).
						Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
				}

				service := services.NewCredentialsCreate(
					repository, serviceShortCodeConsume, serviceSignClaims,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				repository.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
				serviceSignClaims.AssertExpectations(t)
			})
		})
	}
}
