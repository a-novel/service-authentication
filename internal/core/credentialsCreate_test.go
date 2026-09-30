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
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-authentication/v2/internal/config"
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
		resp *core.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsCreateRequest

		daoMock                     *daoMock
		issueTokenMock              *issueTokenMock
		serviceSignClaimsMock       *serviceSignClaimsMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock

		expect     *core.Token
		expectErr  error
		expectRole string
		cleanupErr error
	}{
		{
			name:       "Success/WaitlistUnavailable",
			cleanupErr: dao.ErrWaitlistUnavailable,

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
			name: "Success/RegistrationRole",

			request: &core.CredentialsCreateRequest{
				Email:     "admin@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{Data: []byte(`{"role":"auth:admin"}`)},
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Email:     "admin@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      config.RoleAdmin,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{},

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{Token: "access-token"},
			},

			expect: &core.Token{
				AccessToken:  "access-token",
				RefreshToken: mockUnsignedRefreshToken,
			},
			expectRole: config.RoleAdmin,
		},
		{
			name: "Error/MalformedRegistrationData",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{Data: []byte(`{"role":`)},
			},

			expectErr: core.ErrCredentialsCreateInvalidRegistrationData,
		},
		{
			name: "Error/UnknownRegistrationRole",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{Data: []byte(`{"role":"auth:unknown"}`)},
			},

			expectErr: core.ErrCredentialsCreateInvalidRegistrationData,
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

			ctx := t.Context()
			mockDao := coremocks.NewMockCredentialsCreateDao(t)
			serviceShortCodeConsume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
			serviceSignClaims := coremocks.NewMockCredentialsCreateServiceSignClaims(t)

			waitlist := coremocks.NewMockCredentialsCreateWaitlist(t)
			if testCase.daoMock != nil && testCase.daoMock.err == nil {
				waitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
					Action: "remove", Email: testCase.request.Email,
				}).Return(&dao.WaitlistResult{}, testCase.cleanupErr).Once()
			}

			if testCase.serviceShortCodeConsumeMock != nil {
				shortCode := testCase.serviceShortCodeConsumeMock.resp
				if shortCode == nil && testCase.serviceShortCodeConsumeMock.err == nil {
					shortCode = &core.ShortCode{}
				}

				serviceShortCodeConsume.EXPECT().
					Exec(mock.Anything, &core.ShortCodeConsumeRequest{
						Usage:  core.ShortCodeUsageRegister,
						Target: testCase.request.Email,
						Code:   testCase.request.ShortCode,
					}).
					Return(shortCode, testCase.serviceShortCodeConsumeMock.err)
			}

			if testCase.daoMock != nil {
				expectRole := testCase.expectRole
				if expectRole == "" {
					expectRole = config.RoleUser
				}

				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
						return assert.Equal(t, testCase.request.Email, data.Email) &&
							assert.NotEqual(t, uuid.Nil, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
							assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
							assert.Equal(t, expectRole, data.Role)
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
				mockDao, serviceShortCodeConsume, serviceSignClaims, transactiontest.NewTransactor(), waitlist,
			)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			serviceShortCodeConsume.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			waitlist.AssertExpectations(t)
		})
	}
}

// TestCredentialsCreateIsAtomic keeps sheet cleanup outside the transaction and after its commit,
// even when the client disconnects before cleanup starts.
func TestCredentialsCreateIsAtomic(t *testing.T) {
	t.Parallel()

	errTransaction := errors.New("transaction failed")

	testCases := []struct {
		name      string
		opened    bool
		committed bool
		expectErr error
	}{
		{name: "BeginFails", expectErr: errTransaction},
		{name: "CommitFails", opened: true, expectErr: errTransaction},
		{name: "DisconnectAfterCommit", opened: true, committed: true, expectErr: context.Canceled},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			request := &core.CredentialsCreateRequest{
				Email: "user@provider.com", Password: "secret", ShortCode: "foobarqux",
			}
			mockDao := coremocks.NewMockCredentialsCreateDao(t)
			consume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
			sign := coremocks.NewMockCredentialsCreateServiceSignClaims(t)
			waitlist := coremocks.NewMockCredentialsCreateWaitlist(t)
			transactor := coremocks.NewMockTransactor(t)
			committed := false

			transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					if !testCase.opened {
						return errTransaction
					}

					require.NoError(t, fn(ctx))

					if !testCase.committed {
						return errTransaction
					}

					committed = true

					cancel()

					return nil
				}).Once()

			if testCase.opened {
				consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{
					Usage: core.ShortCodeUsageRegister, Target: request.Email, Code: request.ShortCode,
				}).Return(&core.ShortCode{}, nil).Once()
				mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()
			}

			if testCase.committed {
				waitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{Action: "remove", Email: request.Email}).
					Run(func(cleanupCtx context.Context, _ *dao.WaitlistRequest) {
						require.True(t, committed, "sheet I/O must happen after commit")
						require.ErrorIs(t, ctx.Err(), context.Canceled)
						require.NoError(t, cleanupCtx.Err(), "cleanup must survive client cancellation")
						deadline, ok := cleanupCtx.Deadline()
						require.True(t, ok, "cleanup must remain bounded")
						require.Positive(t, time.Until(deadline))
						require.LessOrEqual(t, time.Until(deadline), 3*time.Second)
					}).Return(&dao.WaitlistResult{}, nil).Once()
				sign.EXPECT().ClaimsSign(mock.Anything, mock.Anything).Return(nil, context.Canceled).Once()
			}

			service := core.NewCredentialsCreate(mockDao, consume, sign, transactor, waitlist)
			resp, err := service.Exec(ctx, request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Nil(t, resp)
			mockDao.AssertExpectations(t)
			consume.AssertExpectations(t)
			sign.AssertExpectations(t)
			waitlist.AssertExpectations(t)
			transactor.AssertExpectations(t)
		})
	}
}
