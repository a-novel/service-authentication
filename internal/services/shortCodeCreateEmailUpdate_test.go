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
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := config.SmtpUrls{UpdateEmail: "update-email-url"}

	type serviceCreateMock struct {
		resp *services.ShortCode
		err  error
	}

	type repositorySelectMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *services.ShortCodeCreateEmailUpdateRequest

		serviceCreateMock    *serviceCreateMock
		repositorySelectMock *repositorySelectMock
		sendMail             bool

		expectErr error
	}{
		{
			name: "Success",

			request: &services.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			repositorySelectMock: &repositorySelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			sendMail: true,
		},
		{
			name: "Error/CreateShortCode",

			request: &services.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			repositorySelectMock: &repositorySelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/EmailAlreadyTaken",

			request: &services.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			repositorySelectMock: &repositorySelectMock{
				err: nil,
			},

			expectErr: dao.ErrCredentialsUpdateEmailAlreadyExists,
		},
		{
			name: "Error/Repository",

			request: &services.ShortCodeCreateEmailUpdateRequest{
				Lang:  config.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			repositorySelectMock: &repositorySelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := servicesmocks.NewMockShortCodeCreateEmailUpdateService(t)
			repositorySelect := servicesmocks.NewMockShortCodeCreateEmailUpdateRepository(t)
			smtpService := servicesmocks.NewMockShortCodeCreateEmailUpdateSmtp(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, &services.ShortCodeCreateRequest{
						Usage:    services.ShortCodeUsageValidateEmail,
						Target:   testCase.request.ID.String(),
						TTL:      config.ShortCodesPresetDefault.Usages[services.ShortCodeUsageValidateEmail].TTL,
						Data:     testCase.request.Email,
						Override: true,
					}).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.repositorySelectMock != nil {
				repositorySelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(nil, testCase.repositorySelectMock.err)
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
								Usages[services.ShortCodeUsageValidateEmail].
								TTL.Hours(),
							"Banner":   assets.BannerBase64,
							"_Purpose": "email-update",
						},
					).
					Return(nil)
			}

			service := services.NewShortCodeCreateEmailUpdate(
				serviceCreate, repositorySelect, smtpService, config.ShortCodesPresetDefault, smtpConfig,
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
			repositorySelect.AssertExpectations(t)
			smtpService.AssertExpectations(t)
		})
	}
}
