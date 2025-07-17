package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
)

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := models.SMTPURLsConfig{UpdatePassword: "update-password-url"}

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type createShortCodeData struct {
		resp *models.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request services.RequestPasswordResetRequest

		selectCredentialsData *selectCredentialsData
		createShortCodeData   *createShortCodeData
		sendMail              bool

		expectErr error
	}{
		{
			name: "Success",

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
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

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			createShortCodeData: &createShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "SelectEmailError",

			request: services.RequestPasswordResetRequest{
				Email: "user@provider.com",
			},

			selectCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockRequestPasswordResetSource(t)

			if testCase.createShortCodeData != nil {
				source.EXPECT().
					CreateShortCode(mock.Anything, services.CreateShortCodeRequest{
						Usage:    models.ShortCodeUsageResetPassword,
						Target:   testCase.selectCredentialsData.resp.ID.String(),
						TTL:      config.ShortCodesPresetDefault.Usages[models.ShortCodeUsageResetPassword].TTL,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.selectCredentialsData != nil {
				source.EXPECT().
					SelectCredentialsByEmail(mock.Anything, testCase.request.Email).
					Return(testCase.selectCredentialsData.resp, testCase.selectCredentialsData.err)
			}

			if testCase.sendMail {
				source.EXPECT().
					SendMail(
						[]string{testCase.request.Email},
						models.Mails.PasswordReset,
						testCase.request.Lang.String(),
						map[string]any{
							"ShortCode": testCase.createShortCodeData.resp.PlainCode,
							"Target":    testCase.selectCredentialsData.resp.ID.String(),
							"URL":       smtpConfig.UpdatePassword,
							"Duration": config.ShortCodesPresetDefault.
								Usages[models.ShortCodeUsageResetPassword].
								TTL.String(),
							"_Purpose": "password-reset",
						},
					).
					Return(nil)
			}

			service := services.NewRequestPasswordResetService(source, config.ShortCodesPresetDefault, smtpConfig)

			resp, err := service.RequestPasswordReset(t.Context(), testCase.request)
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
