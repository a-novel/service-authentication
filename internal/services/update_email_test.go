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

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	testutils "github.com/a-novel/service-authentication/internal/test"
	"github.com/a-novel/service-authentication/models"
)

func TestUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type updateCredentialsEmailData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type consumeShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.UpdateEmailRequest

		consumeShortCodeData       *consumeShortCodeData
		updateCredentialsEmailData *updateCredentialsEmailData

		expect    *services.UpdateEmailResponse
		expectErr error
	}{
		{
			name: "Success",

			request: services.UpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			consumeShortCodeData: &consumeShortCodeData{
				resp: &models.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

			updateCredentialsEmailData: &updateCredentialsEmailData{
				resp: &dao.CredentialsEntity{
					Email: "user@provider.com",
				},
			},

			expect: &services.UpdateEmailResponse{
				NewEmail: "user@provider.com",
			},
		},
		{
			name: "Error/ConsumeShortCode",

			request: services.UpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			consumeShortCodeData: &consumeShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UpdateCredentialsEmail",

			request: services.UpdateEmailRequest{
				UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ShortCode: "shortCode",
			},

			consumeShortCodeData: &consumeShortCodeData{
				resp: &models.ShortCode{
					Data: []byte(`"user@provider.com"`),
				},
			},

			updateCredentialsEmailData: &updateCredentialsEmailData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, testutils.TestDBConfig, func(ctx context.Context, t *testing.T) {
				t.Helper()

				source := servicesmocks.NewMockUpdateEmailSource(t)

				if testCase.consumeShortCodeData != nil {
					source.EXPECT().
						ConsumeShortCode(mock.Anything, services.ConsumeShortCodeRequest{
							Usage:  models.ShortCodeUsageValidateMail,
							Target: testCase.request.UserID.String(),
							Code:   testCase.request.ShortCode,
						}).
						Return(testCase.consumeShortCodeData.resp, testCase.consumeShortCodeData.err)
				}

				if testCase.updateCredentialsEmailData != nil {
					source.EXPECT().
						UpdateCredentialsEmail(
							mock.Anything,
							testCase.request.UserID,
							mock.MatchedBy(func(data dao.UpdateCredentialsEmailData) bool {
								var newEmail string
								err := json.Unmarshal(testCase.consumeShortCodeData.resp.Data, &newEmail)

								return assert.NoError(t, err) &&
									assert.Equal(t, newEmail, data.Email) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Second)
							}),
						).
						Return(testCase.updateCredentialsEmailData.resp, testCase.updateCredentialsEmailData.err)
				}

				service := services.NewUpdateEmailService(source)

				resp, err := service.UpdateEmail(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				source.AssertExpectations(t)
			})
		})
	}
}
