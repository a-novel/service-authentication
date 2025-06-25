package services_test

import (
	"errors"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/stretchr/testify/mock"
	"testing"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestRequestEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

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
						TTL:      config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL,
						Data:     testCase.request.Email,
						Override: true,
					}).
					Return(testCase.createShortCodeData.resp, testCase.createShortCodeData.err)
			}

			if testCase.sendMail {
				source.EXPECT().SMTP(
					mock.Anything,
					mock.MatchedBy(func(req *template.Template) bool {
						return req.Name() == mails.Mails.EmailUpdate.Name()
					}),
					testCase.request.Lang,
					[]string{testCase.request.Email},
					map[string]any{
						"ShortCode": testCase.createShortCodeData.resp.PlainCode,
						"Target":    testCase.request.ID.String(),
						"URL":       config.SMTP.URLs.UpdateEmail,
						"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL.String(),
					},
				).
					Return()
			}

			service := services.NewRequestEmailUpdateService(source)

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
