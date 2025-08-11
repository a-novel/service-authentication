package services_test

import (
	"encoding/base64"
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

func TestRequestRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := models.SMTPURLsConfig{Register: "register-url"}

	type createShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.RequestRegisterRequest

		createShortCodeData *createShortCodeData
		sendMail            bool

		expectErr error
	}{
		{
			name: "Success",

			request: services.RequestRegisterRequest{
				Email: "user@provider.com",
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

			request: services.RequestRegisterRequest{
				Email: "user@provider.com",
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

			source := servicesmocks.NewMockRequestRegisterSource(t)

			if testCase.createShortCodeData != nil {
				source.EXPECT().
					CreateShortCode(mock.Anything, services.CreateShortCodeRequest{
						Usage:    models.ShortCodeUsageRequestRegister,
						Target:   testCase.request.Email,
						TTL:      config.ShortCodesPresetDefault.Usages[models.ShortCodeUsageRequestRegister].TTL,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.sendMail {
				source.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						models.Mails.Register,
						testCase.request.Lang.String(),
						map[string]any{
							"ShortCode": testCase.createShortCodeData.resp.PlainCode,
							"Target":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
							"URL":       smtpConfig.Register,
							"Duration": config.ShortCodesPresetDefault.
								Usages[models.ShortCodeUsageRequestRegister].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "register",
						},
					).
					Return(nil)
			}

			service := services.NewRequestRegisterService(source, config.ShortCodesPresetDefault, smtpConfig)

			resp, err := service.RequestRegister(t.Context(), testCase.request)
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
