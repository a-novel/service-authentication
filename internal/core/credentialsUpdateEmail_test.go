package core_test

import (
	"context"
	"encoding/json"
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

func TestCredentialsUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000010")

	type serviceShortCodeVerifyMock struct {
		resp *core.ShortCode
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.CredentialsUpdateEmailRequest

		serviceShortCodeVerifyMock  *serviceShortCodeVerifyMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock
		daoMock                     *daoMock

		expect    *core.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{
					ID:   shortCodeID,
					Data: []byte(`"user@provider.com"`),
				},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					Email: "user@provider.com",
				},
			},

			expect: &core.Credentials{
				Email: "user@provider.com",
			},
		},
		{
			name: "Error/VerifyShortCode",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			// Another request redeemed the code after this one verified it.
			name: "Error/ConsumeShortCode",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{
					ID:   shortCodeID,
					Data: []byte(`"user@provider.com"`),
				},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: core.ErrShortCodeConsumeInvalid,
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/UpdateCredentialsEmail",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				resp: &core.ShortCode{
					ID:   shortCodeID,
					Data: []byte(`"user@provider.com"`),
				},
			},
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsUpdateEmailDao(t)
				serviceShortCodeVerify := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeVerify(t)
				serviceShortCodeConsume := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)

				if testCase.serviceShortCodeVerifyMock != nil {
					shortCodeVerifyRequest := &core.ShortCodeVerifyRequest{
						Usage:  core.ShortCodeUsageValidateEmail,
						Target: testCase.request.UserID.String(),
						Code:   testCase.request.ShortCode,
					}

					serviceShortCodeVerify.EXPECT().
						Exec(mock.Anything, shortCodeVerifyRequest).
						Return(testCase.serviceShortCodeVerifyMock.resp, testCase.serviceShortCodeVerifyMock.err)
				}

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCodeID}).
						Return(&core.ShortCode{ID: shortCodeID}, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(
							mock.Anything,
							mock.MatchedBy(func(data *dao.CredentialsUpdateEmailRequest) bool {
								var newEmail string

								err := json.Unmarshal(testCase.serviceShortCodeVerifyMock.resp.Data, &newEmail)

								return assert.NoError(t, err) &&
									assert.Equal(t, newEmail, data.Email) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
									assert.Equal(t, testCase.request.UserID, data.ID)
							}),
						).
						Return(testCase.daoMock.resp, testCase.daoMock.err)
				}

				service := core.NewCredentialsUpdateEmail(
					mockDao, serviceShortCodeVerify, serviceShortCodeConsume, transactiontest.NewTransactor(),
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				mockDao.AssertExpectations(t)
				serviceShortCodeVerify.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}

// TestCredentialsUpdateEmailHashing checks the short code, which runs Argon2, before the
// transaction, which holds a pooled connection.
//
//nolint:paralleltest // It closes the process-wide hash gate, which parallel tests share.
func TestCredentialsUpdateEmailHashing(t *testing.T) {
	request := &core.CredentialsUpdateEmailRequest{
		UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ShortCode: "shortCode",
	}
	shortCode := &core.ShortCode{
		ID:   uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		Data: []byte(`"user@provider.com"`),
	}

	mockDao := coremocks.NewMockCredentialsUpdateEmailDao(t)
	verify := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeVerify(t)
	consume := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)
	transactor := coremocks.NewMockTransactor(t)

	verify.EXPECT().Exec(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *core.ShortCodeVerifyRequest) (*core.ShortCode, error) {
			// The real check hashes too.
			lib.DummyCompareArgon2(request.ShortCode)

			return shortCode, nil
		}).Once()
	transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).RunAndReturn(noHashInTx(t)).Once()
	consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).Return(shortCode, nil).Once()
	mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()

	service := core.NewCredentialsUpdateEmail(mockDao, verify, consume, transactor)

	_, err := service.Exec(t.Context(), request)
	require.NoError(t, err)
	mockDao.AssertExpectations(t)
	verify.AssertExpectations(t)
	consume.AssertExpectations(t)
	transactor.AssertExpectations(t)
}
