package services_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestShortCodeCreateRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := config.SmtpUrls{Register: "register-url"}

	type serviceCreateMock struct {
		resp *services.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *services.ShortCodeCreateRegisterRequest

		serviceCreateMock *serviceCreateMock
		sendMail          bool

		expectErr error
	}{
		{
			name: "Success",

			request: &services.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			serviceCreateMock: &serviceCreateMock{
				resp: &services.ShortCode{
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

			request: &services.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := servicesmocks.NewMockShortCodeCreateRegisterService(t)
			smtpService := servicesmocks.NewMockShortCodeCreateRegisterSmtp(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, &services.ShortCodeCreateRequest{
						Usage:    services.ShortCodeUsageRegister,
						Target:   testCase.request.Email,
						TTL:      config.ShortCodesPresetDefault.Usages[services.ShortCodeUsageRegister].TTL,
						Override: true,
					}).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.sendMail {
				smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.Register,
						testCase.request.Lang,
						map[string]any{
							"ShortCode": testCase.serviceCreateMock.resp.PlainCode,
							"Target":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
							"URL":       smtpConfig.Register,
							"Duration": config.ShortCodesPresetDefault.
								Usages[services.ShortCodeUsageRegister].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "register",
						},
					).
					Return(nil)
			}

			service := services.NewShortCodeCreateRegister(
				serviceCreate, smtpService, config.ShortCodesPresetDefault, smtpConfig,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.serviceCreateMock.resp, resp)
			}

			if testCase.sendMail {
				service.Wait()
			}

			serviceCreate.AssertExpectations(t)
			smtpService.AssertExpectations(t)
		})
	}
}
