package core_test

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
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
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

var mailOutboxSmtpConfig = authconfig.SmtpUrls{
	Register:       "register-url",
	UpdateEmail:    "update-email-url",
	UpdatePassword: "update-password-url",
}

// mailOutboxMocks holds the dependencies of a [core.MailOutbox] under test, and the failures its
// report received.
type mailOutboxMocks struct {
	list       *coremocks.MockMailOutboxDaoList
	claim      *coremocks.MockMailOutboxDaoClaim
	renew      *coremocks.MockMailOutboxDaoRenew
	reschedule *coremocks.MockMailOutboxDaoReschedule
	delete     *coremocks.MockMailOutboxDaoDelete
	smtp       *coremocks.MockMailOutboxSMTP

	mu      sync.Mutex
	reports []error
}

func newMailOutbox(t *testing.T, limit int) (*core.MailOutbox, *mailOutboxMocks) {
	t.Helper()

	mocks := &mailOutboxMocks{
		list:       coremocks.NewMockMailOutboxDaoList(t),
		claim:      coremocks.NewMockMailOutboxDaoClaim(t),
		renew:      coremocks.NewMockMailOutboxDaoRenew(t),
		reschedule: coremocks.NewMockMailOutboxDaoReschedule(t),
		delete:     coremocks.NewMockMailOutboxDaoDelete(t),
		smtp:       coremocks.NewMockMailOutboxSMTP(t),
	}

	report := func(_ context.Context, usage string, err error) {
		mocks.mu.Lock()
		defer mocks.mu.Unlock()

		require.NotEmpty(t, usage)

		mocks.reports = append(mocks.reports, err)
	}

	outbox := core.NewMailOutbox(
		mocks.list, mocks.claim, mocks.renew, mocks.reschedule, mocks.delete, mocks.smtp, limit, report,
		authconfig.ShortCodesPresetDefault, mailOutboxSmtpConfig,
	)

	return outbox, mocks
}

// expectEnd expects the entry of shortCodeID to end.
func (mocks *mailOutboxMocks) expectEnd(shortCodeID uuid.UUID) *coremocks.MockMailOutboxDaoDelete_Exec_Call {
	return mocks.delete.EXPECT().
		Exec(mock.Anything, &dao.MailOutboxDeleteRequest{ShortCodeID: shortCodeID}).
		Return(&dao.MailOutboxEntry{ShortCodeID: shortCodeID}, nil)
}

func TestMailOutbox(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := "00000000-0000-0000-0000-000000000002"

	newEntry := func(usage, target string, attempts int, expiresIn time.Duration) *dao.MailOutboxEntry {
		return &dao.MailOutboxEntry{
			ShortCodeID: shortCodeID,
			Usage:       usage,
			Target:      target,
			Recipient:   "user@provider.com",
			Lang:        authconfig.LangFR,
			Attempts:    attempts,
			ExpiresAt:   time.Now().Add(expiresIn),
		}
	}

	register := newEntry(core.ShortCodeUsageRegister, "user@provider.com", 1, 48*time.Hour)
	registerData := map[string]any{
		mails.TemplateVarShortCode: "plain-code",
		mails.TemplateVarTarget:    base64.RawURLEncoding.EncodeToString([]byte("user@provider.com")),
		mails.TemplateVarURL:       mailOutboxSmtpConfig.Register,
		mails.TemplateVarDuration:  authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageRegister].TTL.Hours(),
		mails.TemplateVarBanner:    assets.BannerBase64,
	}

	type sendMock struct {
		template *template.Template
		data     map[string]any
		err      error
		panic    bool
	}

	testCases := []struct {
		name string

		job        *core.MailOutboxJob
		claim      *dao.MailOutboxEntry
		claimErr   error
		renewErr   error
		renewCalls bool
		send       *sendMock

		expectRetry  time.Duration
		expectEnd    bool
		expectReport error
		expectErr    error
	}{
		{
			name: "Success/Register",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: register,
			send:  &sendMock{template: mails.Mails.Register, data: registerData},

			expectEnd: true,
		},
		{
			name: "Success/EmailUpdate",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: newEntry(core.ShortCodeUsageValidateEmail, userID, 1, 48*time.Hour),
			send: &sendMock{
				template: mails.Mails.EmailUpdate,
				data: map[string]any{
					mails.TemplateVarShortCode: "plain-code",
					mails.TemplateVarTarget:    userID,
					mails.TemplateVarSource:    base64.RawURLEncoding.EncodeToString([]byte("user@provider.com")),
					mails.TemplateVarURL:       mailOutboxSmtpConfig.UpdateEmail,
					mails.TemplateVarDuration: authconfig.ShortCodesPresetDefault.
						Usages[core.ShortCodeUsageValidateEmail].TTL.Hours(),
					mails.TemplateVarBanner: assets.BannerBase64,
				},
			},

			expectEnd: true,
		},
		{
			name: "Success/PasswordReset",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: newEntry(core.ShortCodeUsageResetPassword, userID, 1, 2*time.Hour),
			send: &sendMock{
				template: mails.Mails.PasswordReset,
				data: map[string]any{
					mails.TemplateVarShortCode: "plain-code",
					mails.TemplateVarTarget:    userID,
					mails.TemplateVarURL:       mailOutboxSmtpConfig.UpdatePassword,
					mails.TemplateVarDuration: authconfig.ShortCodesPresetDefault.
						Usages[core.ShortCodeUsageResetPassword].TTL.Hours(),
					mails.TemplateVarBanner: assets.BannerBase64,
				},
			},

			expectEnd: true,
		},
		{
			// An entry read back from the outbox carries no plaintext, so the attempt renews the code
			// and sends the new one.
			name: "Success/Renewed",

			job:        &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim:      register,
			renewCalls: true,
			send:       &sendMock{template: mails.Mails.Register},

			expectEnd: true,
		},
		{
			name: "Success/NotClaimed",

			job:      &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claimErr: dao.ErrMailOutboxClaimNotFound,
		},
		{
			// A consumed or superseded code ends its entry without a mail or a report.
			name: "Success/CodeGone",

			job:        &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim:      register,
			renewCalls: true,
			renewErr:   dao.ErrShortCodeRenewNotFound,

			expectEnd: true,
		},
		{
			name: "Error/Transient/FirstRetry",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: register,
			send:  &sendMock{template: mails.Mails.Register, data: registerData, err: errors.Join(errFoo, smtp.ErrTransient)},

			expectRetry: 30 * time.Minute,
			expectErr:   smtp.ErrTransient,
		},
		{
			name: "Error/Transient/SecondRetry",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim: newEntry(core.ShortCodeUsageRegister, "user@provider.com", 2, 48*time.Hour),
			send:  &sendMock{template: mails.Mails.Register, err: errors.Join(errFoo, smtp.ErrTransient)},

			renewCalls:  true,
			expectRetry: time.Hour,
			expectErr:   smtp.ErrTransient,
		},
		{
			name: "Error/Transient/LaterRetries",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim: newEntry(core.ShortCodeUsageRegister, "user@provider.com", 7, 48*time.Hour),
			send:  &sendMock{template: mails.Mails.Register, err: errors.Join(errFoo, smtp.ErrTransient)},

			renewCalls:  true,
			expectRetry: 2 * time.Hour,
			expectErr:   smtp.ErrTransient,
		},
		{
			// The next retry would come after the first code's expiry, so the entry ends.
			name: "Error/Transient/Expiring",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim: newEntry(core.ShortCodeUsageRegister, "user@provider.com", 3, time.Hour),
			send:  &sendMock{template: mails.Mails.Register, err: errors.Join(errFoo, smtp.ErrTransient)},

			renewCalls:   true,
			expectEnd:    true,
			expectReport: core.ErrMailOutboxExpired,
			expectErr:    core.ErrMailOutboxExpired,
		},
		{
			name: "Error/Permanent",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: register,
			send:  &sendMock{template: mails.Mails.Register, data: registerData, err: errors.Join(errFoo, smtp.ErrPermanent)},

			expectEnd:    true,
			expectReport: smtp.ErrPermanent,
			expectErr:    smtp.ErrPermanent,
		},
		{
			// The relay may hold the mail, so another attempt could deliver it twice.
			name: "Error/Uncertain",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: register,
			send:  &sendMock{template: mails.Mails.Register, data: registerData, err: errors.Join(errFoo, smtp.ErrUncertain)},

			expectEnd:    true,
			expectReport: smtp.ErrUncertain,
			expectErr:    smtp.ErrUncertain,
		},
		{
			// An instance killed mid-attempt leaves its entry until the lease ends, past the expiry.
			name: "Error/Expired",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID},
			claim: newEntry(core.ShortCodeUsageRegister, "user@provider.com", 4, -time.Minute),

			expectEnd:    true,
			expectReport: core.ErrMailOutboxExpired,
			expectErr:    core.ErrMailOutboxExpired,
		},
		{
			name: "Error/UnknownUsage",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: newEntry("unknown", "user@provider.com", 1, time.Hour),

			expectEnd:    true,
			expectReport: core.ErrMailOutboxUnknownUsage,
			expectErr:    core.ErrMailOutboxUnknownUsage,
		},
		{
			name: "Error/Claim",

			job:      &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claimErr: errFoo,

			expectErr: errFoo,
		},
		{
			// A panic in a sender would end the process. The entry keeps its lease, and an attempt
			// after the lease retries it.
			name: "Error/Panic",

			job:   &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"},
			claim: register,
			send:  &sendMock{template: mails.Mails.Register, data: registerData, panic: true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			outbox, mocks := newMailOutbox(t, 1)

			before := time.Now()

			mocks.claim.EXPECT().
				Exec(mock.Anything, mock.MatchedBy(func(request *dao.MailOutboxClaimRequest) bool {
					return request.ShortCodeID == shortCodeID &&
						!request.Now.Before(before) &&
						request.LeaseUntil.Equal(request.Now.Add(time.Hour))
				})).
				Return(testCase.claim, testCase.claimErr)

			var renewedHash string

			if testCase.renewCalls {
				ttl := authconfig.ShortCodesPresetDefault.Usages[testCase.claim.Usage].TTL

				mocks.renew.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.ShortCodeRenewRequest) bool {
						return request.ID == shortCodeID && request.Code != "" &&
							request.ExpiresAt.Equal(request.Now.Add(ttl))
					})).
					RunAndReturn(func(_ context.Context, request *dao.ShortCodeRenewRequest) (*dao.ShortCode, error) {
						renewedHash = request.Code

						return &dao.ShortCode{ID: shortCodeID}, testCase.renewErr
					})
			}

			if testCase.send != nil {
				var data any = testCase.send.data
				if testCase.send.data == nil {
					// A renewed code is random, so the mail must carry the code whose hash was stored.
					data = mock.MatchedBy(func(data map[string]any) bool {
						plainCode, _ := data[mails.TemplateVarShortCode].(string)

						return lib.CompareArgon2(plainCode, renewedHash) == nil
					})
				}

				sendMail := mocks.smtp.EXPECT().
					SendMail(smtp.MailUsers{{Email: "user@provider.com"}}, testCase.send.template, authconfig.LangFR, data).
					Return(testCase.send.err)

				if testCase.send.panic {
					sendMail.Run(func(smtp.MailUsers, *template.Template, string, any) { panic("smtp exploded") })
				}
			}

			if testCase.expectRetry > 0 {
				mocks.reschedule.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.MailOutboxRescheduleRequest) bool {
						return request.ShortCodeID == shortCodeID &&
							!request.DueAt.Before(before.Add(testCase.expectRetry)) &&
							!request.DueAt.After(time.Now().Add(testCase.expectRetry))
					})).
					Return(&dao.MailOutboxEntry{ShortCodeID: shortCodeID}, nil)
			}

			if testCase.expectEnd {
				mocks.expectEnd(shortCodeID)
			}

			err := outbox.Deliver(t.Context(), testCase.job)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				require.NoError(t, err)
			}

			if testCase.expectReport == nil {
				require.Empty(t, mocks.reports)
			} else {
				require.Len(t, mocks.reports, 1)
				require.ErrorIs(t, mocks.reports[0], testCase.expectReport)
			}
		})
	}

	// A starting instance reads the pending entries back at once and delivers them with a renewed
	// code each.
	t.Run("Success/Run/Resume", func(t *testing.T) {
		t.Parallel()

		outbox, mocks := newMailOutbox(t, 2)

		ids := []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000011"),
			uuid.MustParse("00000000-0000-0000-0000-000000000012"),
		}

		mocks.list.EXPECT().
			Exec(mock.Anything, mock.MatchedBy(func(request *dao.MailOutboxListRequest) bool { return request.Limit > 0 })).
			Return([]*dao.MailOutboxEntry{{ShortCodeID: ids[0]}, {ShortCodeID: ids[1]}}, nil).
			Once()

		var ended sync.WaitGroup

		for _, id := range ids {
			entry := &dao.MailOutboxEntry{
				ShortCodeID: id,
				Usage:       core.ShortCodeUsageRegister,
				Target:      "user@provider.com",
				Recipient:   "user@provider.com",
				Lang:        authconfig.LangFR,
				Attempts:    2,
				ExpiresAt:   time.Now().Add(time.Hour),
			}

			mocks.claim.EXPECT().
				Exec(mock.Anything, mock.MatchedBy(func(request *dao.MailOutboxClaimRequest) bool {
					return request.ShortCodeID == id
				})).
				Return(entry, nil)
			mocks.renew.EXPECT().
				Exec(mock.Anything, mock.MatchedBy(func(request *dao.ShortCodeRenewRequest) bool { return request.ID == id })).
				Return(&dao.ShortCode{ID: id}, nil)

			ended.Add(1)
			mocks.expectEnd(id).Run(func(context.Context, *dao.MailOutboxDeleteRequest) { ended.Done() })
		}

		mocks.smtp.EXPECT().SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice()

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)

		go func() { done <- outbox.Run(ctx, 5*time.Second) }()

		ended.Wait()
		cancel()

		require.NoError(t, <-done)
		require.Empty(t, mocks.reports)
	})

	// A request hands its new code to the senders, so its first attempt needs no renewal.
	t.Run("Success/Run/Enqueue", func(t *testing.T) {
		t.Parallel()

		outbox, mocks := newMailOutbox(t, 1)

		mocks.list.EXPECT().Exec(mock.Anything, mock.Anything).Return([]*dao.MailOutboxEntry{}, nil).Once()
		mocks.claim.EXPECT().Exec(mock.Anything, mock.Anything).Return(register, nil)
		mocks.smtp.EXPECT().
			SendMail(smtp.MailUsers{{Email: "user@provider.com"}}, mails.Mails.Register, authconfig.LangFR, registerData).
			Return(nil)

		ended := make(chan struct{})

		mocks.expectEnd(shortCodeID).Run(func(context.Context, *dao.MailOutboxDeleteRequest) { close(ended) })

		outbox.Enqueue(t.Context(), &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"})

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)

		go func() { done <- outbox.Run(ctx, 5*time.Second) }()

		<-ended
		cancel()

		require.NoError(t, <-done)
	})

	// A shutdown that cuts an attempt makes its entry due at once, so the next instance retries it
	// instead of waiting out the lease.
	t.Run("Error/Run/Release", func(t *testing.T) {
		t.Parallel()

		outbox, mocks := newMailOutbox(t, 1)

		mocks.list.EXPECT().Exec(mock.Anything, mock.Anything).Return([]*dao.MailOutboxEntry{}, nil).Once()
		mocks.claim.EXPECT().Exec(mock.Anything, mock.Anything).Return(register, nil)

		sending := make(chan struct{})
		unblock := make(chan struct{})

		mocks.smtp.EXPECT().
			SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(smtp.MailUsers, *template.Template, string, any) error {
				close(sending)
				<-unblock

				return nil
			})

		released := time.Now()

		mocks.reschedule.EXPECT().
			Exec(mock.Anything, mock.MatchedBy(func(request *dao.MailOutboxRescheduleRequest) bool {
				return request.ShortCodeID == shortCodeID && !request.DueAt.Before(released) &&
					request.DueAt.Before(time.Now().Add(time.Second))
			})).
			Return(register, nil)

		// The cut attempt finishes after the release, and settles the entry like any other.
		settled := make(chan struct{})

		mocks.expectEnd(shortCodeID).Run(func(context.Context, *dao.MailOutboxDeleteRequest) { close(settled) })

		outbox.Enqueue(t.Context(), &core.MailOutboxJob{ShortCodeID: shortCodeID, PlainCode: "plain-code"})

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)

		go func() { done <- outbox.Run(ctx, time.Second+50*time.Millisecond) }()

		<-sending
		cancel()

		require.ErrorIs(t, <-done, context.DeadlineExceeded)

		close(unblock)
		<-settled
	})
}
