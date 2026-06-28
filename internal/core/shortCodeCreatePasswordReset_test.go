package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

func TestShortCodeCreatePasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := config.SmtpUrls{UpdatePassword: "update-password-url"}

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreatePasswordResetRequest

		daoMock           *daoMock
		serviceCreateMock *serviceCreateMock
		sendMail          bool

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			serviceCreateMock: &serviceCreateMock{
				resp: &core.ShortCode{
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

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectEmailError",

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
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

			serviceCreate := coremocks.NewMockShortCodeCreatePasswordResetService(t)
			mockDao := coremocks.NewMockShortCodeCreatePasswordResetDao(t)
			smtpService := coremocks.NewMockShortCodeCreatePasswordResetSmtp(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, &core.ShortCodeCreateRequest{
						Usage:    core.ShortCodeUsageResetPassword,
						Target:   testCase.daoMock.resp.ID.String(),
						TTL:      config.ShortCodesPresetDefault.Usages[core.ShortCodeUsageResetPassword].TTL,
						Override: true,
					}).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.sendMail {
				smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.PasswordReset,
						testCase.request.Lang,
						map[string]any{
							"ShortCode": testCase.serviceCreateMock.resp.PlainCode,
							"Target":    testCase.daoMock.resp.ID.String(),
							"URL":       smtpConfig.UpdatePassword,
							"Duration": config.ShortCodesPresetDefault.
								Usages[core.ShortCodeUsageResetPassword].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "password-reset",
						},
					).
					Return(nil)
			}

			service := core.NewShortCodeCreatePasswordReset(
				serviceCreate, mockDao, smtpService, config.ShortCodesPresetDefault, smtpConfig,
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
			mockDao.AssertExpectations(t)
			smtpService.AssertExpectations(t)
		})
	}
}
