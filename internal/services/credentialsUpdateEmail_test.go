package services_test

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

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestCredentialsUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		resp *services.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsUpdateEmailRequest

		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock
		repositoryMock              *repositoryMock

		expect    *services.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &services.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					Email: "user@provider.com",
				},
			},

			expect: &services.Credentials{
				Email: "user@provider.com",
			},
		},
		{
			name: "Error/ConsumeShortCode",

			request: &services.CredentialsUpdateEmailRequest{
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

			request: &services.CredentialsUpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				resp: &services.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

			repositoryMock: &repositoryMock{
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

				repository := servicesmocks.NewMockCredentialsUpdateEmailRepository(t)
				serviceShortCodeConsume := servicesmocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &services.ShortCodeConsumeRequest{
							Usage:  services.ShortCodeUsageValidateEmail,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.serviceShortCodeConsumeMock.resp, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.repositoryMock != nil {
					repository.EXPECT().
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
						Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
				}

				service := services.NewCredentialsUpdateEmail(repository, serviceShortCodeConsume)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				repository.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
			})
		})
	}
}
