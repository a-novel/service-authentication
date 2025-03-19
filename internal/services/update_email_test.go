package services_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestUpdateEmail(t *testing.T) { //nolint:paralleltest
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

	for _, testCase := range testCases { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			ctx, err := pgctx.NewContext(t.Context(), nil)
			require.NoError(t, err)

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
							err = json.Unmarshal(testCase.consumeShortCodeData.resp.Data, &newEmail)

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
	}
}
