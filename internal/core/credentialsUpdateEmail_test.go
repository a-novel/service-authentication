package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"
	"github.com/a-novel-kit/golib/smtp"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

func TestCredentialsUpdateEmail(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	type serviceShortCodeVerifyMock struct {
		resp *core.ShortCode
		err  error
	}

	type serviceShortCodeConsumeMock struct {
		err error
	}

	validRequest := &core.CredentialsUpdateEmailRequest{UserID: userID, ShortCode: "shortCode", Lang: authconfig.LangFR}
	verified := &serviceShortCodeVerifyMock{
		resp: &core.ShortCode{ID: shortCodeID, Data: []byte(`"user@provider.com"`)},
	}
	previous := &daoMock{resp: &dao.Credentials{ID: userID, Email: "former@provider.com"}}

	testCases := []struct {
		name string

		request *core.CredentialsUpdateEmailRequest

		serviceShortCodeVerifyMock  *serviceShortCodeVerifyMock
		closeMailDelivery           bool
		daoSelectMock               *daoMock
		serviceShortCodeConsumeMock *serviceShortCodeConsumeMock
		daoMock                     *daoMock
		// sendNotice expects the notice to the former address.
		sendNotice bool

		expect    *core.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			serviceShortCodeVerifyMock:  verified,
			daoSelectMock:               previous,
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			daoMock: &daoMock{
				resp: &dao.Credentials{
					Email: "user@provider.com",
				},
			},
			sendNotice: true,

			expect: &core.Credentials{
				Email: "user@provider.com",
			},
		},
		{
			// The notice goes in the language the request names.
			name: "Error/NoLang",

			request: &core.CredentialsUpdateEmailRequest{UserID: userID, ShortCode: "shortCode"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/VerifyShortCode",

			request: validRequest,

			serviceShortCodeVerifyMock: &serviceShortCodeVerifyMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			// Without room for the notice, the change does not apply.
			name: "Error/MailDeliveryUnavailable",

			request: validRequest,

			serviceShortCodeVerifyMock: verified,
			closeMailDelivery:          true,

			expectErr: core.ErrMailDeliveryUnavailable,
		},
		{
			name: "Error/SelectCredentials",

			request: validRequest,

			serviceShortCodeVerifyMock: verified,
			daoSelectMock:              &daoMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// Another request redeemed the code after this one verified it.
			name: "Error/ConsumeShortCode",

			request: validRequest,

			serviceShortCodeVerifyMock: verified,
			daoSelectMock:              previous,
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{
				err: core.ErrShortCodeConsumeInvalid,
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/UpdateCredentialsEmail",

			request: validRequest,

			serviceShortCodeVerifyMock:  verified,
			daoSelectMock:               previous,
			serviceShortCodeConsumeMock: &serviceShortCodeConsumeMock{},
			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				mockDao := coremocks.NewMockCredentialsUpdateEmailDao(t)
				mockDaoSelect := coremocks.NewMockCredentialsUpdateEmailDaoSelect(t)
				serviceShortCodeVerify := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeVerify(t)
				serviceShortCodeConsume := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)
				smtpService := coremocks.NewMockMailDeliverySMTP(t)

				mailDelivery := core.NewMailDelivery(smtpService, 1)
				if testCase.closeMailDelivery {
					mailDelivery.Close()
				}

				if testCase.serviceShortCodeVerifyMock != nil {
					shortCodeVerifyRequest := &core.ShortCodeVerifyRequest{
						Usage:  core.ShortCodeUsageValidateEmail,
						Target: testCase.request.UserID.String(),
						Code:   testCase.request.ShortCode,
					}

					serviceShortCodeVerify.EXPECT().
						Exec(mock.Anything, shortCodeVerifyRequest).
						Return(testCase.serviceShortCodeVerifyMock.resp, testCase.serviceShortCodeVerifyMock.err)
				}

				if testCase.daoSelectMock != nil {
					mockDaoSelect.EXPECT().
						Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
						Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
				}

				if testCase.serviceShortCodeConsumeMock != nil {
					serviceShortCodeConsume.EXPECT().
						Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCodeID}).
						Return(&core.ShortCode{ID: shortCodeID}, testCase.serviceShortCodeConsumeMock.err)
				}

				if testCase.daoMock != nil {
					mockDao.EXPECT().
						Exec(
							mock.Anything,
							mock.MatchedBy(func(data *dao.CredentialsUpdateEmailRequest) bool {
								var newEmail string

								err := json.Unmarshal(testCase.serviceShortCodeVerifyMock.resp.Data, &newEmail)

								return assert.NoError(t, err) &&
									assert.Equal(t, newEmail, data.Email) &&
									assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
									assert.Equal(t, testCase.request.UserID, data.ID)
							}),
						).
						Return(testCase.daoMock.resp, testCase.daoMock.err)
				}

				if testCase.sendNotice {
					smtpService.EXPECT().
						SendMail(
							smtp.MailUsers{{Email: "former@provider.com"}},
							mails.Mails.EmailChanged,
							testCase.request.Lang,
							nil,
						).
						Return(nil)
				}

				service := core.NewCredentialsUpdateEmail(
					mockDao, mockDaoSelect, serviceShortCodeVerify, serviceShortCodeConsume, mailDelivery,
					transactiontest.NewTransactor(),
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				require.NoError(t, mailDelivery.Wait(ctx))

				mockDao.AssertExpectations(t)
				mockDaoSelect.AssertExpectations(t)
				serviceShortCodeVerify.AssertExpectations(t)
				serviceShortCodeConsume.AssertExpectations(t)
				smtpService.AssertExpectations(t)
			})
		})
	}
}

// TestCredentialsUpdateEmailHashing checks the short code, which runs Argon2, before the
// transaction, which holds a pooled connection.
//
//nolint:paralleltest // It closes the process-wide hash gate, which parallel tests share.
func TestCredentialsUpdateEmailHashing(t *testing.T) {
	request := &core.CredentialsUpdateEmailRequest{
		UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ShortCode: "shortCode",
		Lang:      authconfig.LangEN,
	}
	shortCode := &core.ShortCode{
		ID:   uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		Data: []byte(`"user@provider.com"`),
	}

	mockDao := coremocks.NewMockCredentialsUpdateEmailDao(t)
	mockDaoSelect := coremocks.NewMockCredentialsUpdateEmailDaoSelect(t)
	verify := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeVerify(t)
	consume := coremocks.NewMockCredentialsUpdateEmailServiceShortCodeConsume(t)
	transactor := coremocks.NewMockTransactor(t)
	smtpService := coremocks.NewMockMailDeliverySMTP(t)
	mailDelivery := core.NewMailDelivery(smtpService, 1)

	verify.EXPECT().Exec(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *core.ShortCodeVerifyRequest) (*core.ShortCode, error) {
			// The real check hashes too.
			lib.DummyCompareArgon2(request.ShortCode)

			return shortCode, nil
		}).Once()
	transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).RunAndReturn(noHashInTx(t)).Once()
	mockDaoSelect.EXPECT().Exec(mock.Anything, mock.Anything).
		Return(&dao.Credentials{Email: "former@provider.com"}, nil).Once()
	consume.EXPECT().Exec(mock.Anything, &core.ShortCodeConsumeRequest{ID: shortCode.ID}).Return(shortCode, nil).Once()
	mockDao.EXPECT().Exec(mock.Anything, mock.Anything).Return(&dao.Credentials{}, nil).Once()
	smtpService.EXPECT().SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	service := core.NewCredentialsUpdateEmail(mockDao, mockDaoSelect, verify, consume, mailDelivery, transactor)

	_, err := service.Exec(t.Context(), request)
	require.NoError(t, err)

	waitForMailDelivery(t, mailDelivery)
	mockDao.AssertExpectations(t)
	mockDaoSelect.AssertExpectations(t)
	verify.AssertExpectations(t)
	consume.AssertExpectations(t)
	transactor.AssertExpectations(t)
	smtpService.AssertExpectations(t)
}
