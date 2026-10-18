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

	type serviceShortCodeConsumeMock struct {
		resp *core.ShortCode
		err  error
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

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{},
			},

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

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{},
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ConsumesShortCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: errFoo,
			},

			expectErr: errFoo,
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

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{},
			},

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
				serviceShortCodeConsume := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeConsume(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{
							Usage:  core.ShortCodeUsageResetPassword,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.serviceShortCodeConsumeMock.resp, testCase.serviceShortCodeConsumeMock.err)
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
					mockDao, daoCredentialsSelect, serviceShortCodeConsume, transactiontest.NewTransactor(),
				)

				_, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				mockDao.AssertExpectations(t)
				daoCredentialsSelect.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}

// TestCredentialsUpdatePasswordChecksProofFirst hashes no new password until the short code or the
// current password checks out, so a failed check costs at most the single Argon2 run of the check.
//
//nolint:paralleltest // It closes the process-wide hash gate, which parallel tests share.
func TestCredentialsUpdatePasswordChecksProofFirst(t *testing.T) {
	closeArgon2Gate(t)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name string

		request *core.CredentialsUpdatePasswordRequest

		expectErr error
	}{
		{
			name: "Error/ShortCode",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", ShortCode: "short-code",
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/CurrentPassword",

			request: &core.CredentialsUpdatePasswordRequest{
				UserID: userID, Password: "new-password", CurrentPassword: "password",
			},

			expectErr: dao.ErrCredentialsSelectNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Checking the current password hashes too, so the case stops at the lookup before it.
			daoCredentialsSelect := coremocks.NewMockCredentialsUpdatePasswordDaoCredentialsSelect(t)
			if testCase.request.CurrentPassword != "" {
				daoCredentialsSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(nil, dao.ErrCredentialsSelectNotFound).Once()
			}

			serviceShortCodeConsume := coremocks.NewMockCredentialsUpdatePasswordServiceShortCodeConsume(t)
			if testCase.request.ShortCode != "" {
				serviceShortCodeConsume.EXPECT().
					Exec(mock.Anything, &core.ShortCodeConsumeRequest{
						Usage:  core.ShortCodeUsageResetPassword,
						Target: userID.String(),
						Code:   testCase.request.ShortCode,
					}).
					Return(nil, core.ErrShortCodeConsumeInvalid).Once()
			}

			service := core.NewCredentialsUpdatePassword(
				coremocks.NewMockCredentialsUpdatePasswordDao(t),
				daoCredentialsSelect,
				serviceShortCodeConsume,
				transactiontest.NewTransactor(),
			)

			err := requireNoHash(t, func() error {
				_, err := service.Exec(t.Context(), testCase.request)

				return err
			})
			require.ErrorIs(t, err, testCase.expectErr)
			daoCredentialsSelect.AssertExpectations(t)
			serviceShortCodeConsume.AssertExpectations(t)
		})
	}
}
