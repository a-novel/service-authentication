package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestCredentialsUpdatePassword(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordArgon2ed, err := lib.GenerateArgon2(passwordRaw, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	type serviceShortCodeConsumeMock struct {
		resp *services.ShortCode
		err  error
	}

	type repositoryCredentialsSelectMock struct {
		resp *dao.Credentials
		err  error
	}

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsUpdatePasswordRequest

		serviceShortCodeConsumeMock     *serviceShortCodeConsumeMock
		repositoryCredentialsSelectMock *repositoryCredentialsSelectMock
		repositoryMock                  *repositoryMock

		expectErr error
	}{
		{
			name: "Success/ShorCode",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &services.ShortCode{},
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{},
			},
		},
		{
			name: "Success/CurrentPassword",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: passwordRaw,
			},

			repositoryCredentialsSelectMock: &repositoryCredentialsSelectMock{
				resp: &dao.Credentials{
					Password: passwordArgon2ed,
				},
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{},
			},
		},
		{
			name: "Error/UploadCredentials",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "new-password",
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &services.ShortCode{},
			},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ConsumesShortCode",

			request: &services.CredentialsUpdatePasswordRequest{
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

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: passwordRaw,
			},

			repositoryCredentialsSelectMock: &repositoryCredentialsSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/WrongPassword",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:        "new-password",
				CurrentPassword: "fake-password",
			},

			repositoryCredentialsSelectMock: &repositoryCredentialsSelectMock{
				resp: &dao.Credentials{
					Password: passwordArgon2ed,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/MissingShortCodeAndCurrentPassword",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/PasswordTooShort",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "abc", // 3 characters, minimum is 4
				ShortCode: "short-code",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Success/PasswordAtMinLength",

			request: &services.CredentialsUpdatePasswordRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Password:  "abcd", // exactly 4 characters
				ShortCode: "short-code",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &services.ShortCode{},
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				repository := servicesmocks.NewMockCredentialsUpdatePasswordRepository(t)
				repositoryCredentialsSelect := servicesmocks.NewMockCredentialsUpdatePasswordRepositoryCredentialsSelect(t)
				serviceShortCodeConsume := servicesmocks.NewMockCredentialsUpdatePasswordServiceShortCodeConsume(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &services.ShortCodeConsumeRequest{
							Usage:  services.ShortCodeUsageResetPassword,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.serviceShortCodeConsumeMock.resp, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.repositoryCredentialsSelectMock != nil {
					repositoryCredentialsSelect.EXPECT().
						Exec(mock.Anything, &dao.CredentialsSelectRequest{
							ID: testCase.request.UserID,
						}).
						Return(
							testCase.repositoryCredentialsSelectMock.resp,
							testCase.repositoryCredentialsSelectMock.err,
						)
				}

				if testCase.repositoryMock != nil {
					repository.EXPECT().
						Exec(
							mock.Anything,
							mock.MatchedBy(func(data *dao.CredentialsUpdatePasswordRequest) bool {
								return assert.NoError(t, lib.CompareArgon2(testCase.request.Password, data.Password)) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Minute) &&
									assert.Equal(t, testCase.request.UserID, data.ID)
							}),
						).
						Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
				}

				service := services.NewCredentialsUpdatePassword(
					repository, repositoryCredentialsSelect, serviceShortCodeConsume,
				)

				_, err = service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				repository.AssertExpectations(t)
				repositoryCredentialsSelect.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}
