package core_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"text/template"
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

func TestShortCodeCreateRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := config.SmtpUrls{Register: "register-url"}

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	type daoSelectMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreateRegisterRequest

		serviceCreateMock *serviceCreateMock
		daoSelectMock     *daoSelectMock
		sendMail          bool
		sendMailPanic     bool

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
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
			name: "Success/SendMailPanic",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
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

			sendMail:      true,
			sendMailPanic: true,
		},
		{
			name: "Error/CreateShortCode",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/EmailAlreadyTaken",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoSelectMock: &daoSelectMock{
				err: nil,
			},

			expectErr: dao.ErrCredentialsInsertAlreadyExists,
		},
		{
			name: "Error/Dao",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
			},

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := coremocks.NewMockShortCodeCreateRegisterService(t)
			daoSelect := coremocks.NewMockShortCodeCreateRegisterDao(t)
			smtpService := coremocks.NewMockShortCodeCreateRegisterSmtp(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, &core.ShortCodeCreateRequest{
						Usage:    core.ShortCodeUsageRegister,
						Target:   testCase.request.Email,
						TTL:      config.ShortCodesPresetDefault.Usages[core.ShortCodeUsageRegister].TTL,
						Override: true,
					}).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.sendMail {
				sendMail := smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.Register,
						testCase.request.Lang,
						map[string]any{
							"ShortCode": testCase.serviceCreateMock.resp.PlainCode,
							"Target":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
							"URL":       smtpConfig.Register,
							"Duration": config.ShortCodesPresetDefault.
								Usages[core.ShortCodeUsageRegister].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "register",
						},
					).
					Return(nil)

				if testCase.sendMailPanic {
					// The delivery runs on a detached goroutine; a panic it fails to absorb ends
					// the test binary, so surviving to Wait is the assertion.
					sendMail.Run(func(smtp.MailUsers, *template.Template, string, any) {
						panic("mail delivery exploded")
					})
				}
			}

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(nil, testCase.daoSelectMock.err)
			}

			service := core.NewShortCodeCreateRegister(
				serviceCreate, daoSelect, smtpService, config.ShortCodesPresetDefault, smtpConfig,
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
			daoSelect.AssertExpectations(t)
			smtpService.AssertExpectations(t)
		})
	}
}
