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

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	smtpConfig := authconfig.SMTPURLs{UpdateEmail: "update-email-url"}

	const currentPassword = "current password"

	currentPasswordHash, err := lib.GenerateArgon2(currentPassword)
	require.NoError(t, err)

	request := &core.ShortCodeCreateEmailUpdateRequest{
		CurrentPassword: currentPassword,
		Lang:            authconfig.LangFR,
		Email:           "user@provider.com",
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}
	shortCode := &core.ShortCode{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Usage:     "test-usage",
		Target:    "test-target",
		Data:      []byte("test-data"),
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		PlainCode: "abcdef123456",
	}

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

		request *core.ShortCodeCreateEmailUpdateRequest

		// invalid skips every dependency: the request fails validation.
		invalid bool
		// credentialsErr fails the account lookup.
		credentialsErr    error
		serviceCreateMock *serviceCreateMock
		daoExistMock      *daoExistMock
		sendMail          bool
		closeMailDelivery bool

		expectErr error
	}{
		{
			name: "Success",

			request: request,

			serviceCreateMock: &serviceCreateMock{resp: shortCode},

			daoExistMock: &daoExistMock{},

			sendMail: true,
		},
		{
			// Stores the same code as Success, so the pending change reads the same, but mails nothing.
			name: "Success/EmailTaken",

			request: request,

			serviceCreateMock: &serviceCreateMock{resp: shortCode},

			daoExistMock: &daoExistMock{exists: true},
		},
		{
			name: "Error/MailDeliveryUnavailable",

			request: request,

			daoExistMock: &daoExistMock{},

			closeMailDelivery: true,
			expectErr:         core.ErrMailDeliveryUnavailable,
		},
		{
			// A saturated mailer must fail a taken address as it fails a free one.
			name: "Error/MailDeliveryUnavailable/EmailTaken",

			request: request,

			daoExistMock: &daoExistMock{exists: true},

			closeMailDelivery: true,
			expectErr:         core.ErrMailDeliveryUnavailable,
		},
		{
			name: "Error/CreateShortCode",

			request: request,

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			daoExistMock: &daoExistMock{},

			expectErr: errFoo,
		},
		{
			name: "Error/Dao",

			request: request,

			daoExistMock: &daoExistMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/NoCurrentPassword",

			request: changed(*request, func(request *core.ShortCodeCreateEmailUpdateRequest) { request.CurrentPassword = "" }),

			invalid:   true,
			expectErr: core.ErrInvalidRequest,
		},
		{
			// A session alone does not move the account to another address.
			name: "Error/WrongCurrentPassword",

			request: changed(*request, func(request *core.ShortCodeCreateEmailUpdateRequest) {
				request.CurrentPassword = "wrong password"
			}),

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/SelectCredentials",

			request: request,

			credentialsErr: errFoo,
			expectErr:      errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := coremocks.NewMockShortCodeCreateEmailUpdateServiceShortCodeCreate(t)
			daoExist := coremocks.NewMockShortCodeCreateEmailUpdateDaoCredentialsExist(t)
			daoCredentials := coremocks.NewMockShortCodeCreateEmailUpdateDaoCredentials(t)

			if !testCase.invalid {
				credentials := &dao.Credentials{ID: testCase.request.ID, Password: currentPasswordHash}
				if testCase.credentialsErr != nil {
					credentials = nil
				}

				daoCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: testCase.request.ID}).
					Return(credentials, testCase.credentialsErr)
			}

			smtpService := coremocks.NewMockMailDeliverySMTP(t)

			mailDelivery := core.NewMailDelivery(smtpService, 1)
			if testCase.closeMailDelivery {
				mailDelivery.Close()
			}

			if testCase.serviceCreateMock != nil {
				shortCodeCreateRequest := &core.ShortCodeCreateRequest{
					Usage:  core.ShortCodeUsageValidateEmail,
					Target: testCase.request.ID.String(),
					TTL:    authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL,
					Data:   testCase.request.Email,
				}

				serviceCreate.EXPECT().
					Exec(mock.Anything, shortCodeCreateRequest).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoExistMock != nil {
				daoExist.EXPECT().
					Exec(mock.Anything, &dao.CredentialsExistRequest{Email: testCase.request.Email}).
					Return(testCase.daoExistMock.exists, testCase.daoExistMock.err)
			}

			if testCase.sendMail {
				mailData := mails.Link{
					ShortCode: testCase.serviceCreateMock.resp.PlainCode,
					Target:    testCase.request.ID.String(),
					Source:    base64.RawURLEncoding.EncodeToString([]byte(testCase.request.Email)),
					URL:       smtpConfig.UpdateEmail,
					Duration:  authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL.Hours(),
				}

				smtpService.EXPECT().
					SendMail(
						smtp.MailUsers{{Email: testCase.request.Email}},
						mails.Mails.EmailUpdate,
						testCase.request.Lang,
						mailData,
					).
					Return(nil)
			}

			service := core.NewShortCodeCreateEmailUpdate(
				serviceCreate, daoExist, daoCredentials, mailDelivery, authconfig.ShortCodesPresetDefault, smtpConfig,
			)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			waitForMailDelivery(t, mailDelivery)
		})
	}
}
