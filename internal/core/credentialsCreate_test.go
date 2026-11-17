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

// TestCredentialsCreate keeps one success path through the session the account opens, which
// TestTokenCreate covers in full.
func TestCredentialsCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// A new account consents to no location, so its session records the user agent alone.
	request := &core.CredentialsCreateRequest{
		Email:     "user@provider.com",
		Password:  "password-2",
		ShortCode: "short-code",
		Visitor:   core.Visitor{UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris"},
	}
	shortCode := &core.ShortCode{ID: uuid.MustParse("00000000-0000-0000-0000-000000000010")}
	credentials := &dao.Credentials{
		ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Email: "user@provider.com",
		Role:  authconfig.RoleUser,
	}

	type respMock[T any] struct {
		resp T
		err  error
	}

	type errMock struct {
		err error
	}

	// transactionMock runs the transaction's body, unless it fails to begin with beginErr. A body that
	// succeeds then fails to commit with commitErr, or commits, and the client leaves on disconnect.
	type transactionMock struct {
		beginErr   error
		commitErr  error
		disconnect bool
	}

	type testCase struct {
		name string

		request *core.CredentialsCreateRequest

		verifyMock      *respMock[*core.ShortCode]
		transactionMock *transactionMock
		consumeMock     *errMock
		daoMock         *respMock[*dao.Credentials]
		waitlistMock    *errMock
		signRefreshMock *errMock
		// sessionOpened signs the access token, and records and bounds the session, all successfully.
		sessionOpened bool

		expect     *core.Token
		expectErr  error
		expectRole string
	}

	success := &testCase{
		name: "Success",

		request: request,

		verifyMock:      &respMock[*core.ShortCode]{resp: shortCode},
		transactionMock: &transactionMock{},
		consumeMock:     &errMock{},
		daoMock:         &respMock[*dao.Credentials]{resp: credentials},
		waitlistMock:    &errMock{},
		signRefreshMock: &errMock{},
		sessionOpened:   true,

		expect:     &core.Token{AccessToken: "access-token", RefreshToken: mockUnsignedRefreshToken},
		expectRole: authconfig.RoleUser,
	}

	testCases := []*testCase{
		success,
		changed(*success, func(variant *testCase) {
			variant.name = "Success/WaitlistUnavailable"
			variant.waitlistMock = &errMock{err: dao.ErrWaitlistUnavailable}
		}),
		changed(*success, func(variant *testCase) {
			variant.name = "Success/RegistrationRole"
			variant.verifyMock = &respMock[*core.ShortCode]{
				resp: changed(*shortCode, func(shortCode *core.ShortCode) { shortCode.Data = []byte(`{"role":"auth:admin"}`) }),
			}
			variant.daoMock = &respMock[*dao.Credentials]{
				resp: changed(*credentials, func(credentials *dao.Credentials) { credentials.Role = authconfig.RoleAdmin }),
			}
			variant.expectRole = authconfig.RoleAdmin
		}),
		changed(*success, func(variant *testCase) {
			variant.name = "Success/PasswordAtMinLength"
			variant.request = changed(*request, func(request *core.CredentialsCreateRequest) { request.Password = "abcd" })
		}),
		{
			name: "Error/PasswordTooShort",

			request: changed(*request, func(request *core.CredentialsCreateRequest) { request.Password = "abc" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/VerifyShortCode",

			request: request,

			verifyMock: &respMock[*core.ShortCode]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/MalformedRegistrationData",

			request: request,

			verifyMock: &respMock[*core.ShortCode]{resp: &core.ShortCode{Data: []byte(`{"role":`)}},

			expectErr: core.ErrCredentialsCreateInvalidRegistrationData,
		},
		{
			name: "Error/UnknownRegistrationRole",

			request: request,

			verifyMock: &respMock[*core.ShortCode]{resp: &core.ShortCode{Data: []byte(`{"role":"auth:unknown"}`)}},

			expectErr: core.ErrCredentialsCreateInvalidRegistrationData,
		},
		{
			name: "Error/BeginTransaction",

			request: request,

			verifyMock:      &respMock[*core.ShortCode]{resp: shortCode},
			transactionMock: &transactionMock{beginErr: errFoo},

			expectErr: errFoo,
		},
		{
			// Another request redeemed the code while this one hashed the password.
			name: "Error/ConsumeShortCode",

			request: request,

			verifyMock:      &respMock[*core.ShortCode]{resp: shortCode},
			transactionMock: &transactionMock{},
			consumeMock:     &errMock{err: core.ErrShortCodeInvalid},

			expectErr: core.ErrShortCodeInvalid,
		},
		{
			name: "Error/CreateCredentials",

			request: request,

			verifyMock:      &respMock[*core.ShortCode]{resp: shortCode},
			transactionMock: &transactionMock{},
			consumeMock:     &errMock{},
			daoMock:         &respMock[*dao.Credentials]{err: errFoo},

			expectErr:  errFoo,
			expectRole: authconfig.RoleUser,
		},
		{
			// A failed commit leaves the waitlist untouched.
			name: "Error/CommitTransaction",

			request: request,

			verifyMock:      &respMock[*core.ShortCode]{resp: shortCode},
			transactionMock: &transactionMock{commitErr: errFoo},
			consumeMock:     &errMock{},
			daoMock:         &respMock[*dao.Credentials]{resp: credentials},

			expectErr:  errFoo,
			expectRole: authconfig.RoleUser,
		},
		changed(*success, func(variant *testCase) {
			// The client leaves once the account exists, and the waitlist cleanup still runs.
			variant.name = "Error/DisconnectAfterCommit"
			variant.transactionMock = &transactionMock{disconnect: true}
			variant.signRefreshMock = &errMock{err: context.Canceled}
			variant.sessionOpened = false
			variant.expect, variant.expectErr = nil, context.Canceled
		}),
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
			waitlist := coremocks.NewMockCredentialsCreateDaoWaitlist(t)
			daoLocationConsent := coremocks.NewMockCredentialsCreateDaoLocationConsent(t)
			daoSessionUpsert := coremocks.NewMockCredentialsCreateDaoSessionUpsert(t)
			daoSessionRevokeOldest := coremocks.NewMockCredentialsCreateDaoSessionRevokeOldest(t)
			daoSessionTrim := coremocks.NewMockCredentialsCreateDaoSessionTrim(t)

			var (
				transactions int
				committed    bool
			)

			transactor := transactorFunc(func(ctx context.Context, fn func(context.Context) error) error {
				transactions++
				transaction := lo.FromPtr(testCase.transactionMock)

				if transaction.beginErr != nil {
					return transaction.beginErr
				}

				err := fn(ctx)
				if err != nil {
					return err
				}

				if transaction.commitErr != nil {
					return transaction.commitErr
				}

				committed = true

				if transaction.disconnect {
					cancel()
				}

				return nil
			})

			if testCase.verifyMock != nil {
				shortCodeVerifyRequest := &core.ShortCodeVerifyRequest{
					Usage:  core.ShortCodeUsageRegister,
					Target: testCase.request.Email,
					Code:   testCase.request.ShortCode,
				}

				serviceShortCodeVerify.EXPECT().
					Exec(mock.Anything, shortCodeVerifyRequest).
					Return(testCase.verifyMock.resp, testCase.verifyMock.err)
			}

			if testCase.consumeMock != nil {
				serviceShortCodeConsume.EXPECT().
					Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).
					Return(testCase.consumeMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.CredentialsInsertRequest) bool {
						return assert.Equal(t, testCase.request.Email, data.Email) &&
							assert.NotEqual(t, uuid.Nil, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
							assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
							assert.Equal(t, testCase.expectRole, data.Role)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.waitlistMock != nil {
				// The cleanup runs after the commit, on a bounded context the client cannot cancel.
				waitlist.EXPECT().
					Exec(mock.Anything, &dao.WaitlistRequest{Action: "remove", Email: testCase.request.Email}).
					Run(func(cleanupCtx context.Context, _ *dao.WaitlistRequest) {
						require.True(t, committed, "sheet I/O must happen after commit")
						require.NoError(t, cleanupCtx.Err(), "cleanup must survive client cancellation")

						deadline, ok := cleanupCtx.Deadline()
						require.True(t, ok, "cleanup must remain bounded")
						require.Positive(t, time.Until(deadline))
						require.LessOrEqual(t, time.Until(deadline), 3*time.Second)
					}).
					Return(&dao.WaitlistResult{}, testCase.waitlistMock.err).
					Once()
			}

			if testCase.signRefreshMock != nil {
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuthRefresh,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(session.RefreshClaimsForm{UserID: credentials.ID})),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: mockUnsignedRefreshToken}, testCase.signRefreshMock.err)
			}

			if testCase.sessionOpened {
				accessTokenClaims := session.Claims{
					UserID:         &credentials.ID,
					Roles:          []string{testCase.daoMock.resp.Role},
					RefreshTokenID: mockUnsignedJTI,
					SessionID:      mockUnsignedJTI,
				}
				claimsSignRequest := &servicejsonkeys.ClaimsSignRequest{
					Usage:   servicejsonkeys.KeyUsageAuth,
					Payload: lo.Must(grpcf.MarshalJSONAsAny(accessTokenClaims)),
				}
				sessionRequest := &dao.SessionUpsertRequest{
					ID:        uuid.MustParse(mockUnsignedJTI),
					UserID:    credentials.ID,
					UserAgent: testCase.request.Visitor.UserAgent,
					ExpiresAt: time.Unix(0, 0),
				}

				serviceSignClaims.EXPECT().
					ClaimsSign(mock.Anything, claimsSignRequest).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: "access-token"}, nil)
				daoSessionUpsert.EXPECT().Exec(mock.Anything, matchSessionUpsert(t, sessionRequest)).Return(nil)
				daoSessionRevokeOldest.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil)
				daoSessionTrim.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil)
			}

			service := core.NewCredentialsCreate(
				mockDao,
				daoLocationConsent,
				daoSessionUpsert,
				daoSessionRevokeOldest,
				daoSessionTrim,
				serviceShortCodeVerify,
				serviceShortCodeConsume,
				serviceSignClaims,
				transactor,
				waitlist,
				time.Minute,
			)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)
			require.Equal(t, lo.Ternary(testCase.transactionMock != nil, 1, 0), transactions)
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

			verifyErr: core.ErrShortCodeInvalid,

			expectErr: core.ErrShortCodeInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockDao := coremocks.NewMockCredentialsCreateDao(t)
			verify := coremocks.NewMockCredentialsCreateServiceShortCodeVerify(t)
			consume := coremocks.NewMockCredentialsCreateServiceShortCodeConsume(t)
			sign := coremocks.NewMockCredentialsCreateServiceSignClaims(t)
			waitlist := coremocks.NewMockCredentialsCreateDaoWaitlist(t)
			daoLocationConsent := coremocks.NewMockCredentialsCreateDaoLocationConsent(t)
			daoSessionUpsert := coremocks.NewMockCredentialsCreateDaoSessionUpsert(t)
			daoSessionRevokeOldest := coremocks.NewMockCredentialsCreateDaoSessionRevokeOldest(t)
			daoSessionTrim := coremocks.NewMockCredentialsCreateDaoSessionTrim(t)

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
				consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).
					Return(nil).Once()
				mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()
				waitlist.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.WaitlistResult{}, nil).Once()
				sign.EXPECT().ClaimsSign(mock.Anything, mock.Anything).
					Return(&servicejsonkeys.ClaimsSignResponse{Token: mockUnsignedRefreshToken}, nil).Twice()
				daoSessionUpsert.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil).Once()
				daoSessionRevokeOldest.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil).Once()
				daoSessionTrim.EXPECT().Exec(mock.Anything, mock.Anything).Return(nil).Once()
			}

			service := core.NewCredentialsCreate(
				mockDao, daoLocationConsent, daoSessionUpsert, daoSessionRevokeOldest, daoSessionTrim, verify, consume, sign,
				noHashInTx(t), waitlist, time.Minute,
			)
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
		})
	}
}
