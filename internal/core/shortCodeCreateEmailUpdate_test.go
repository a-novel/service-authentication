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

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := authconfig.SmtpUrls{UpdateEmail: "update-email-url"}

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	type daoSelectMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreateEmailUpdateRequest

		serviceCreateMock *serviceCreateMock
		daoSelectMock     *daoSelectMock
		sendMail          bool
		sendMailPanic     bool
		closeMailDelivery bool

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			sendMail: true,
		},
		{
			name: "Success/SendMailPanic",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			sendMail:      true,
			sendMailPanic: true,
		},
		{
			// Stores the same code as Success, so the pending change reads the same, but mails nothing.
			name: "Success/EmailTaken",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			daoSelectMock: &daoSelectMock{
				err: nil,
			},
		},
		{
			name: "Error/MailDeliveryUnavailable",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			closeMailDelivery: true,
			expectErr:         core.ErrMailDeliveryUnavailable,
		},
		{
			// A saturated mailer must fail a taken address as it fails a free one.
			name: "Error/MailDeliveryUnavailable/EmailTaken",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			daoSelectMock: &daoSelectMock{
				err: nil,
			},

			closeMailDelivery: true,
			expectErr:         core.ErrMailDeliveryUnavailable,
		},
		{
			name: "Error/CreateShortCode",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Dao",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			serviceCreate := coremocks.NewMockShortCodeCreateEmailUpdateService(t)
			daoSelect := coremocks.NewMockShortCodeCreateEmailUpdateDao(t)
			smtpService := coremocks.NewMockMailDeliverySMTP(t)

			mailDelivery := core.NewMailDelivery(smtpService, 1)
			if testCase.closeMailDelivery {
				mailDelivery.Close()
			}

			if testCase.serviceCreateMock != nil {
				shortCodeCreateRequest := &core.ShortCodeCreateRequest{
					Usage:    core.ShortCodeUsageValidateEmail,
					Target:   testCase.request.ID.String(),
					TTL:      authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL,
					Data:     testCase.request.Email,
					Override: true,
				}

				serviceCreate.EXPECT().
					Exec(mock.Anything, shortCodeCreateRequest).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(nil, testCase.daoSelectMock.err)
			}

			if testCase.sendMail {
				mailData := map[string]any{
					"ShortCode": testCase.serviceCreateMock.resp.PlainCode,
					"Target":    testCase.request.ID.String(),
					"Source":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
					"URL":       smtpConfig.UpdateEmail,
					"Duration":  authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL.Hours(),
				}

				sendMail := smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.EmailUpdate,
						testCase.request.Lang,
						mailData,
					).
					Return(nil)

				if testCase.sendMailPanic {
					// An unabsorbed panic ends the test binary; surviving to Wait is the assertion.
					sendMail.Run(func(smtp.MailUsers, *template.Template, string, any) {
						panic("mail delivery exploded")
					})
				}
			}

			service := core.NewShortCodeCreateEmailUpdate(
				serviceCreate, daoSelect, mailDelivery, authconfig.ShortCodesPresetDefault, smtpConfig,
			)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.serviceCreateMock.resp, resp)
			}

			waitForMailDelivery(t, mailDelivery)

			serviceCreate.AssertExpectations(t)
			daoSelect.AssertExpectations(t)
			smtpService.AssertExpectations(t)
		})
	}
}
