package core_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
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

func TestShortCodeCreateRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ShortCodeCreateRegisterRequest{
		Lang:  authconfig.LangFR,
		Email: "user@provider.com",
		Role:  authconfig.RoleUser,
	}

	smtpConfig := authconfig.SMTPURLs{Register: "register-url"}

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	type daoExistMock struct {
		exists bool
		err    error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreateRegisterRequest

		serviceCreateMock *serviceCreateMock
		daoExistMock      *daoExistMock
		sendMail          bool
		closeMailDelivery bool

		expectErr error
	}{
		{
			name: "Success",

			request: request,

			daoExistMock: &daoExistMock{},

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
			name: "Error/InvalidLang",

			request: changed(*request, func(request *core.ShortCodeCreateRegisterRequest) { request.Lang = "xx" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MailDeliveryUnavailable",

			request: request,

			daoExistMock: &daoExistMock{},

			closeMailDelivery: true,
			expectErr:         core.ErrMailDeliveryUnavailable,
		},
		{
			name: "Error/CreateShortCode",

			request: request,

			daoExistMock: &daoExistMock{},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/EmailAlreadyTaken",

			request: request,

			daoExistMock: &daoExistMock{exists: true},

			expectErr: core.ErrCredentialsAlreadyExists,
		},
		{
			name: "Error/Dao",

			request: request,

			daoExistMock: &daoExistMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := coremocks.NewMockShortCodeCreateRegisterServiceShortCodeCreate(t)
			daoExist := coremocks.NewMockShortCodeCreateRegisterDaoCredentialsExist(t)
			smtpService := coremocks.NewMockMailDeliverySMTP(t)

			mailDelivery := core.NewMailDelivery(smtpService, 1)
			if testCase.closeMailDelivery {
				mailDelivery.Close()
			}

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *core.ShortCodeCreateRequest) bool {
						data, err := json.Marshal(request.Data)

						return err == nil &&
							request.Usage == core.ShortCodeUsageRegister &&
							request.Target == testCase.request.Email &&
							request.TTL == authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageRegister].TTL &&
							string(data) == `{"role":"`+testCase.request.Role+`"}`
					})).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.sendMail {
				mailData := mails.Link{
					ShortCode: testCase.serviceCreateMock.resp.PlainCode,
					Target:    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
					URL:       smtpConfig.Register,
					Duration:  authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageRegister].TTL.Hours(),
				}

				smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.Register,
						testCase.request.Lang,
						mailData,
					).
					Return(nil)
			}

			if testCase.daoExistMock != nil {
				daoExist.EXPECT().
					Exec(mock.Anything, &dao.CredentialsExistRequest{Email: testCase.request.Email}).
					Return(testCase.daoExistMock.exists, testCase.daoExistMock.err)
			}

			service := core.NewShortCodeCreateRegister(
				serviceCreate, daoExist, mailDelivery, authconfig.ShortCodesPresetDefault, smtpConfig,
			)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			waitForMailDelivery(t, mailDelivery)
		})
	}
}
