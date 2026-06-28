package core_test

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
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := config.SmtpUrls{UpdateEmail: "update-email-url"}

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

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
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
			name: "Error/CreateShortCode",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
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
			name: "Error/EmailAlreadyTaken",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			daoSelectMock: &daoSelectMock{
				err: nil,
			},

			expectErr: dao.ErrCredentialsUpdateEmailAlreadyExists,
		},
		{
			name: "Error/Dao",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
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
			smtpService := coremocks.NewMockShortCodeCreateEmailUpdateSmtp(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, &core.ShortCodeCreateRequest{
						Usage:    core.ShortCodeUsageValidateEmail,
						Target:   testCase.request.ID.String(),
						TTL:      config.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL,
						Data:     testCase.request.Email,
						Override: true,
					}).
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
				smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.EmailUpdate,
						testCase.request.Lang,
						map[string]any{
							"ShortCode": testCase.serviceCreateMock.resp.PlainCode,
							"Target":    testCase.request.ID.String(),
							"Source":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
							"URL":       smtpConfig.UpdateEmail,
							"Duration": config.ShortCodesPresetDefault.
								Usages[core.ShortCodeUsageValidateEmail].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "email-update",
						},
					).
					Return(nil)
			}

			service := core.NewShortCodeCreateEmailUpdate(
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
