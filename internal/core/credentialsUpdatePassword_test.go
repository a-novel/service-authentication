package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestCredentialsUpdatePassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordArgon2ed, err := lib.GenerateArgon2(passwordRaw, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000010")

	type serviceShortCodeVerifyMock struct {
		resp *core.ShortCode
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		err error
	}

	type daoCredentialsSelectMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsUpdatePasswordRequest

		serviceShortCodeVerifyMock  *serviceShortCodeVerifyMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock
		daoCredentialsSelectMock    *daoCredentialsSelectMock
		daoMock                     *daoMock

		expectErr error
	}{
		{
			name: "Success/ShorCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{ID: shortCodeID},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{},
			},
		},
		{
			name: "Success/CurrentPassword",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: passwordRaw,
			},

			daoCredentialsSelectMock: &daoCredentialsSelectMock{
				resp: &dao.Credentials{
					Password: passwordArgon2ed,
				},
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{},
			},
		},
		{
			name: "Error/UploadCredentials",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{ID: shortCodeID},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/VerifyShortCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
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

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{ID: shortCodeID},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: core.ErrShortCodeConsumeInvalid,
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/SelectCredentials",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: passwordRaw,
			},

			daoCredentialsSelectMock: &daoCredentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/WrongPassword",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: "fake-password",
			},

			daoCredentialsSelectMock: &daoCredentialsSelectMock{
				resp: &dao.Credentials{
					Password: passwordArgon2ed,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/MissingShortCodeAndCurrentPassword",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/PasswordTooShort",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "abc", // 3 characters, minimum is 4
				ShortCode: "short-code",
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Success/PasswordAtMinLength",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "abcd", // exactly 4 characters
				ShortCode: "short-code",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{ID: shortCodeID},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsUpdatePasswordDao(t)
				daoCredentialsSelect := coremocks.NewMockCredentialsUpdatePasswordDaoCredentialsSelect(t)
				serviceShortCodeVerify := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeVerify(t)
				serviceShortCodeConsume := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeConsume(t)

				if testCase.serviceShortCodeVerifyMock != nil {
					serviceShortCodeVerify.EXPECT().
						Exec(mock.Anything, &core.ShortCodeVerifyRequest{
							Usage:  core.ShortCodeUsageResetPassword,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.serviceShortCodeVerifyMock.resp, testCase.serviceShortCodeVerifyMock.err)
				}

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCodeID}).
						Return(&core.ShortCode{ID: shortCodeID}, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.daoCredentialsSelectMock != nil {
					daoCredentialsSelect.EXPECT().
						Exec(mock.Anything, &dao.CredentialsSelectRequest{
							ID: testCase.request.UserID,
						}).
						Return(
							testCase.daoCredentialsSelectMock.resp,
							testCase.daoCredentialsSelectMock.err,
						)
				}

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(
							mock.Anything,
							mock.MatchedBy(func(data *dao.CredentialsUpdatePasswordRequest) bool {
								return assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
									assert.Equal(t, testCase.request.UserID, data.ID)
							}),
						).
						Return(testCase.daoMock.resp, testCase.daoMock.err)
				}

				service := core.NewCredentialsUpdatePassword(
					mockDao,
					daoCredentialsSelect,
					serviceShortCodeVerify,
					serviceShortCodeConsume,
					transactiontest.NewTransactor(),
				)

				_, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				mockDao.AssertExpectations(t)
				daoCredentialsSelect.AssertExpectations(t)
				serviceShortCodeVerify.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}

// TestCredentialsUpdatePasswordHashing keeps every Argon2 run out of the transaction, which holds a
// pooled connection, and hashes no new password until the short code or current password checks out.
//
//nolint:paralleltest // It closes the process-wide hash gate, which parallel tests share.
func TestCredentialsUpdatePasswordHashing(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	shortCode := &core.ShortCode{ID: uuid.MustParse("00000000-0000-0000-0000-000000000010")}

	currentPassword, err := lib.GenerateArgon2("password", lib.Argon2ParamsDefault)
	require.NoError(t, err)

	testCases := []struct {
		name string

		request *core.CredentialsUpdatePasswordRequest

		verifyErr error
		selectErr error

		expectErr error
	}{
		{
			name: "Success/ShortCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", ShortCode: "short-code",
			},
		},
		{
			name: "Success/CurrentPassword",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", CurrentPassword: "password",
			},
		},
		{
			name: "Error/VerifyShortCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", ShortCode: "short-code",
			},

			verifyErr: core.ErrShortCodeConsumeInvalid,

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			// Checking the current password hashes too, so the case stops at the lookup before it.
			name: "Error/SelectCredentials",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", CurrentPassword: "password",
			},

			selectErr: dao.ErrCredentialsSelectNotFound,

			expectErr: dao.ErrCredentialsSelectNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockDao := coremocks.NewMockCredentialsUpdatePasswordDao(t)
			daoCredentialsSelect := coremocks.NewMockCredentialsUpdatePasswordDaoCredentialsSelect(t)
			verify := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeVerify(t)
			consume := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeConsume(t)
			transactor := coremocks.NewMockTransactor(t)

			if testCase.request.ShortCode != "" {
				verify.EXPECT().Exec(mock.Anything, mock.Anything).
					RunAndReturn(func(context.Context, *core.ShortCodeVerifyRequest) (*core.ShortCode, error) {
						if testCase.verifyErr != nil {
							return nil, testCase.verifyErr
						}

						// The real check hashes too.
						lib.DummyCompareArgon2(testCase.request.ShortCode)

						return shortCode, nil
					}).Once()
			}

			if testCase.request.CurrentPassword != "" {
				credentials := &dao.Credentials{Password: currentPassword}
				if testCase.selectErr != nil {
					credentials = nil
				}

				daoCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(credentials, testCase.selectErr).Once()
			}

			if testCase.expectErr == nil {
				transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).RunAndReturn(noHashInTx(t)).Once()
				mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()

				if testCase.request.ShortCode != "" {
					consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).
						Return(shortCode, nil).Once()
				}
			}

			service := core.NewCredentialsUpdatePassword(mockDao, daoCredentialsSelect, verify, consume, transactor)
			exec := func() error {
				_, err := service.Exec(t.Context(), testCase.request)

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
			daoCredentialsSelect.AssertExpectations(t)
			verify.AssertExpectations(t)
			consume.AssertExpectations(t)
			transactor.AssertExpectations(t)
		})
	}
}
