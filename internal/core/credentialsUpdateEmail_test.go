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

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		resp *core.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsUpdateEmailRequest

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

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

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
			name: "Error/ConsumeShortCode",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdateCredentialsEmail",

			request: &core.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &core.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsUpdateEmailDao(t)
				serviceShortCodeConsume := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{
							Usage:  core.ShortCodeUsageValidateEmail,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.serviceShortCodeConsumeMock.resp, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(
							mock.Anything,
							mock.MatchedBy(func(data *dao.CredentialsUpdateEmailRequest) bool {
								var newEmail string

								err := json.Unmarshal(testCase.serviceShortCodeConsumeMock.resp.Data, &newEmail)

								return assert.NoError(t, err) &&
									assert.Equal(t, newEmail, data.Email) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
									assert.Equal(t, testCase.request.UserID, data.ID)
							}),
						).
						Return(testCase.daoMock.resp, testCase.daoMock.err)
				}

				service := core.NewCredentialsUpdateEmail(
					mockDao, serviceShortCodeConsume, transactiontest.NewTransactor(),
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				mockDao.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}
