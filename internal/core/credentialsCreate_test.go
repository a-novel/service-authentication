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

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestCredentialsCreate(t *testing.T) {
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

	type serviceShortCodeVerifyMock struct {
		resp *core.ShortCode
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		err error
	}

	type transactionMock struct {
		beginErr  error
		commitErr error
		// disconnect cancels the request once the transaction commits.
		disconnect bool
	}

	type waitlistMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.CredentialsCreateRequest

		daoMock                     *daoMock
		issueTokenMock              *issueTokenMock
		serviceSignClaimsMock       *serviceSignClaimsMock
		serviceShortCodeVerifyMock  *serviceShortCodeVerifyMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock
		transactionMock             *transactionMock
		waitlistMock                *waitlistMock

		expect     *core.Token
		expectErr  error
		expectRole string
	}{
		{
			name: "Success/WaitlistUnavailable",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},
			waitlistMock:                &waitlistMock{err: dao.ErrWaitlistUnavailable},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      authconfig.RoleUser,
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

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{Data: []byte(`{"role":"auth:admin"}`)},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},
			waitlistMock:                &waitlistMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Email:     "admin@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      authconfig.RoleAdmin,
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
			expectRole: authconfig.RoleAdmin,
		},
		{
			name: "Error/MalformedRegistrationData",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
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

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{Data: []byte(`{"role":"auth:unknown"}`)},
			},

			expectErr: core.ErrCredentialsCreateInvalidRegistrationData,
		},
		{
			name: "Error/VerifyShortCode",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			// Another request redeemed the code while this one hashed the password.
			name: "Error/ConsumeShortCode",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: core.ErrShortCodeConsumeInvalid,
			},
			transactionMock: &transactionMock{},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/CreateCredentials",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},

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

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},
			waitlistMock:                &waitlistMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      authconfig.RoleUser,
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

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},
			waitlistMock:                &waitlistMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      authconfig.RoleUser,
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

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{},
			waitlistMock:                &waitlistMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "abcd-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      authconfig.RoleUser,
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
			name: "Error/BeginTransaction",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{},
			transactionMock:            &transactionMock{beginErr: errFoo},

			expectErr: errFoo,
		},
		{
			// A failed commit leaves the waitlist untouched.
			name: "Error/CommitTransaction",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{commitErr: errFoo},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email: "user@provider.com",
					Role:  authconfig.RoleUser,
				},
			},

			expectErr: errFoo,
		},
		{
			// The client leaves once the account exists, and the waitlist cleanup still runs.
			name: "Error/DisconnectAfterCommit",

			request: &core.CredentialsCreateRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock:  &serviceShortCodeVerifyMock{},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			transactionMock:             &transactionMock{disconnect: true},
			waitlistMock:                &waitlistMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email: "user@provider.com",
					Role:  authconfig.RoleUser,
				},
			},

			serviceSignClaimsMock: &serviceSignClaimsMock{err: context.Canceled},

			expectErr: context.Canceled,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			mockDao := coremocks.NewMockCredentialsCreateDao(t)
			serviceShortCodeVerify := coremocks.NewMockCredentialsCreateServiceShortCodeVerify(t)
			serviceShortCodeConsume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
			serviceSignClaims := coremocks.NewMockCredentialsCreateServiceSignClaims(t)
			transactor := coremocks.NewMockTransactor(t)
			waitlist := coremocks.NewMockCredentialsCreateWaitlist(t)
			committed := false

			if testCase.transactionMock != nil {
				transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						if testCase.transactionMock.beginErr != nil {
							return testCase.transactionMock.beginErr
						}

						err := fn(ctx)
						if err != nil {
							return err
						}

						if testCase.transactionMock.commitErr != nil {
							return testCase.transactionMock.commitErr
						}

						committed = true

						if testCase.transactionMock.disconnect {
							cancel()
						}

						return nil
					}).Once()
			}

			if testCase.waitlistMock != nil {
				// The cleanup runs after the commit, on a bounded context the client cannot cancel.
				waitlist.EXPECT().Exec(mock.Anything, &dao.WaitlistRequest{
					Action: "remove", Email: testCase.request.Email,
				}).
					Run(func(cleanupCtx context.Context, _ *dao.WaitlistRequest) {
						require.True(t, committed, "sheet I/O must happen after commit")
						require.NoError(t, cleanupCtx.Err(), "cleanup must survive client cancellation")

						deadline, ok := cleanupCtx.Deadline()
						require.True(t, ok, "cleanup must remain bounded")
						require.Positive(t, time.Until(deadline))
						require.LessOrEqual(t, time.Until(deadline), 3*time.Second)
					}).
					Return(&dao.WaitlistResult{}, testCase.waitlistMock.err).Once()
			}

			shortCode := &core.ShortCode{ID: uuid.MustParse("00000000-0000-0000-0000-000000000010")}

			if testCase.serviceShortCodeVerifyMock != nil {
				if testCase.serviceShortCodeVerifyMock.resp != nil {
					shortCode = testCase.serviceShortCodeVerifyMock.resp
				}

				shortCodeVerifyRequest := &core.ShortCodeVerifyRequest{
					Usage:  core.ShortCodeUsageRegister,
					Target: testCase.request.Email,
					Code:   testCase.request.ShortCode,
				}

				serviceShortCodeVerify.EXPECT().
					Exec(mock.Anything, shortCodeVerifyRequest).
					Return(shortCode, testCase.serviceShortCodeVerifyMock.err)
			}

			if testCase.serviceShortCodeConsumeMock != nil {
				serviceShortCodeConsume.EXPECT().
					Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).
					Return(shortCode, testCase.serviceShortCodeConsumeMock.err)
			}

			if testCase.daoMock != nil {
				expectRole := testCase.expectRole
				if expectRole == "" {
					expectRole = authconfig.RoleUser
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
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage: servicejsonkeys.KeyUsageAuthRefresh,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(session.RefreshClaimsForm{
						UserID: testCase.daoMock.resp.ID,
					})),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(
						&servicejsonkeys.ClaimsSignResponse{
							Token: mockUnsignedRefreshToken,
						},
						testCase.serviceSignClaimsMock.err,
					)
			}

			if testCase.issueTokenMock != nil {
				accessTokenClaims := session.Claims{
					UserID:         &testCase.daoMock.resp.ID,
					Roles:          []string{testCase.daoMock.resp.Role},
					RefreshTokenID: mockUnsignedJTI,
					SessionID:      mockUnsignedJTI,
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			service := core.NewCredentialsCreate(
				mockDao,
				serviceShortCodeVerify,
				serviceShortCodeConsume,
				serviceSignClaims,
				transactor,
				waitlist,
			)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			serviceShortCodeVerify.AssertExpectations(t)
			serviceShortCodeConsume.AssertExpectations(t)
			serviceSignClaims.AssertExpectations(t)
			transactor.AssertExpectations(t)
			waitlist.AssertExpectations(t)
		})
	}
}

// TestCredentialsCreateHashing keeps every Argon2 run out of the transaction, which holds a pooled
// connection, and hashes no password for a wrong short code.
//
//nolint:paralleltest // It closes the process-wide hash gate, which parallel tests share.
func TestCredentialsCreateHashing(t *testing.T) {
	request := &core.CredentialsCreateRequest{
		Email: "user@provider.com", Password: "secret", ShortCode: "foobarqux",
	}
	shortCode := &core.ShortCode{ID: uuid.MustParse("00000000-0000-0000-0000-000000000010")}

	testCases := []struct {
		name string

		verifyErr error

		expectErr error
	}{
		{name: "Success"},
		{
			name: "Error/VerifyShortCode",

			verifyErr: core.ErrShortCodeConsumeInvalid,

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockDao := coremocks.NewMockCredentialsCreateDao(t)
			verify := coremocks.NewMockCredentialsCreateServiceShortCodeVerify(t)
			consume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
			sign := coremocks.NewMockCredentialsCreateServiceSignClaims(t)
			waitlist := coremocks.NewMockCredentialsCreateWaitlist(t)
			transactor := coremocks.NewMockTransactor(t)

			verify.EXPECT().Exec(mock.Anything, mock.Anything).
				RunAndReturn(func(context.Context, *core.ShortCodeVerifyRequest) (*core.ShortCode, error) {
					if testCase.verifyErr != nil {
						return nil, testCase.verifyErr
					}

					// The real check hashes too.
					lib.DummyCompareArgon2(request.ShortCode)

					return shortCode, nil
				}).Once()

			if testCase.expectErr == nil {
				transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).RunAndReturn(noHashInTx(t)).Once()
				consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).
					Return(shortCode, nil).Once()
				mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()
				waitlist.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.WaitlistResult{}, nil).Once()
				sign.EXPECT().ClaimsSign(mock.Anything, mock.Anything).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: mockUnsignedRefreshToken}, nil).Twice()
			}

			service := core.NewCredentialsCreate(mockDao, verify, consume, sign, transactor, waitlist)
			exec := func() error {
				_, err := service.Exec(t.Context(), request)

				return err
			}

			// A failed check hashes nothing; a passing one hashes before the transaction.
			var err error
			if testCase.expectErr != nil {
				err = requireNoHash(t, exec)
			} else {
				err = exec()
			}

			require.ErrorIs(t, err, testCase.expectErr)
			mockDao.AssertExpectations(t)
			verify.AssertExpectations(t)
			consume.AssertExpectations(t)
			sign.AssertExpectations(t)
			waitlist.AssertExpectations(t)
			transactor.AssertExpectations(t)
		})
	}
}
