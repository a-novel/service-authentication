package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
	"github.com/a-novel/service-authentication/models/mails/assets"
)

func TestRequestEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := models.SMTPURLsConfig{UpdateEmail: "update-email-url"}

	type createShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.RequestEmailUpdateRequest

		createShortCodeData *createShortCodeData
		sendMail            bool

		expectErr error
	}{
		{
			name: "Success",

			request: services.RequestEmailUpdateRequest{
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			createShortCodeData: &createShortCodeData{
				resp: &models.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					PlainCode: "abcdef123456",
				},
			},

			sendMail: true,
		},
		{
			name: "CreateShortCodeError",

			request: services.RequestEmailUpdateRequest{
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			createShortCodeData: &createShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockRequestEmailUpdateSource(t)

			if testCase.createShortCodeData != nil {
				source.EXPECT().
					CreateShortCode(mock.Anything, services.CreateShortCodeRequest{
						Usage:    models.ShortCodeUsageValidateMail,
						Target:   testCase.request.ID.String(),
						TTL:      config.ShortCodesPresetDefault.Usages[models.ShortCodeUsageValidateMail].TTL,
						Data:     testCase.request.Email,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.sendMail {
				source.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						models.Mails.EmailUpdate,
						testCase.request.Lang.String(),
						map[string]any{
							"ShortCode": testCase.createShortCodeData.resp.PlainCode,
							"Target":    testCase.request.ID.String(),
							"URL":       smtpConfig.UpdateEmail,
							"Duration": config.ShortCodesPresetDefault.
								Usages[models.ShortCodeUsageValidateMail].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "email-update",
						},
					).
					Return(nil)
			}

			service := services.NewRequestEmailUpdateService(source, config.ShortCodesPresetDefault, smtpConfig)

			resp, err := service.RequestEmailUpdate(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.createShortCodeData.resp, resp)
			}

			if testCase.sendMail {
				service.Wait()
			}

			source.AssertExpectations(t)
		})
	}
}
