package services_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestRequestRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

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
						TTL:      config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.sendMail {
				source.EXPECT().SMTP(
					mock.Anything,
					mock.MatchedBy(func(req *template.Template) bool {
						return req.Name() == mails.Mails.Register.Name()
					}),
					testCase.request.Lang,
					[]string{testCase.request.Email},
					map[string]any{
						"ShortCode": testCase.createShortCodeData.resp.PlainCode,
						"Target":    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
						"URL":       config.SMTP.URLs.Register,
						"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL.String(),
					},
				).
					Return()
			}

			service := services.NewRequestRegisterService(source)

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
