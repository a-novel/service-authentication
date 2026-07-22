package core_test

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

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestCredentialsCreateRequest(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type issueTokenMock struct {
		resp *servicejsonkeys.ClaimsSignResponse
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

		request *core.CredentialsCreateRequest

		daoMock                     *daoMock
		issueTokenMock              *issueTokenMock
		serviceSignClaimsMock       *serviceSignClaimsMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock

		expect    *core.Token
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
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
			name: "Error/ConsumeShortCode",

			request: &core.CredentialsCreateRequest{
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

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueToken",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
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

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
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
		{
			name: "Error/PasswordTooShort",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "abc", // 3 characters, minimum is 4
				ShortCode: "short-code",
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Success/PasswordAtMinLength",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "abcd", // exactly 4 characters
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "abcd-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      config.RoleUser,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{},

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
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsCreateDao(t)
				serviceShortCodeConsume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
				serviceSignClaims := coremocks.NewMockCredentialsCreateServiceSignClaims(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{
							Usage:  core.ShortCodeUsageRegister,
							Target: testCase.request.Email,
							Code:   testCase.request.ShortCode,
						}).
						Return(nil, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
							return assert.Equal(t, testCase.request.Email, data.Email) &&
								assert.NotEqual(t, uuid.Nil, data.ID) &&
								assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
								assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
								assert.Equal(t, config.RoleUser, data.Role)
						})).
						Return(
							testCase.daoMock.resp,
							testCase.daoMock.err,
						)
				}

				if testCase.serviceSignClaimsMock != nil {
					serviceSignClaims.EXPECT().
						ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
							Usage: servicejsonkeys.KeyUsageAuthRefresh,
							Payload: lo.Must(grpcf.MarshalJSONAsAny(core.RefreshTokenClaimsForm{
								UserID: testCase.daoMock.resp.ID,
							})),
						}).
						Return(
							&servicejsonkeys.ClaimsSignResponse{
								Token: mockUnsignedRefreshToken,
							},
							testCase.serviceSignClaimsMock.err,
						)
				}

				if testCase.issueTokenMock != nil {
					serviceSignClaims.EXPECT().
						ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
							Usage: servicejsonkeys.KeyUsageAuth,
							Payload: lo.Must(grpcf.MarshalJSONAsAny(core.AccessTokenClaims{
								UserID:         &testCase.daoMock.resp.ID,
								Roles:          []string{testCase.daoMock.resp.Role},
								RefreshTokenID: mockUnsignedJTI,
							})),
						}).
						Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
				}

				service := core.NewCredentialsCreate(
					mockDao, serviceShortCodeConsume, serviceSignClaims, transactiontest.NewTransactor(),
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				mockDao.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
				serviceSignClaims.AssertExpectations(t)
			})
		})
	}
}

// TestCredentialsCreateIsAtomic proves the two writes run inside one unit of
// work rather than merely appearing to.
//
// It fails against the code this replaced: that opened a transaction, handed the
// callback the surrounding context, and let both writes resolve the connection
// pool and commit on their own — so consuming the short code survived a failed
// credentials insert, and the user could not retry with the link they were sent.
// A transactor that refuses to open reproduces that boundary exactly: if the
// writes are inside the scope, neither dependency is ever reached.
func TestCredentialsCreateIsAtomic(t *testing.T) {
	t.Parallel()

	errNoTransaction := errors.New("transaction unavailable")

	mockDao := coremocks.NewMockCredentialsCreateDao(t)
	serviceShortCodeConsume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
	serviceSignClaims := coremocks.NewMockCredentialsCreateServiceSignClaims(t)

	transactor := transactiontest.NewFailingTransactor(errNoTransaction)

	service := core.NewCredentialsCreate(mockDao, serviceShortCodeConsume, serviceSignClaims, transactor)

	resp, err := service.Exec(t.Context(), &core.CredentialsCreateRequest{
		Email:     "user@provider.com",
		Password:  "secret",
		ShortCode: "foobarqux",
	})
	require.ErrorIs(t, err, errNoTransaction)
	require.Nil(t, resp)

	require.Equal(t, 1, transactor.Calls(), "the operation must open exactly one unit of work")

	// No expectations were registered on either mock, so mockery fails the test if
	// anything reached them outside the scope that never opened.
	mockDao.AssertExpectations(t)
	serviceShortCodeConsume.AssertExpectations(t)
	serviceSignClaims.AssertExpectations(t)
}
