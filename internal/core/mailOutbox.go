package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

const (
	// mailOutboxLease keeps a claimed entry from other attempts. It outlasts a send whose relay stalls
	// at one step, 10 minutes at most, so only an instance killed mid-attempt leaves its entry waiting.
	mailOutboxLease = time.Hour
	// mailOutboxPollInterval is how often a running outbox reads the due entries back.
	mailOutboxPollInterval = time.Minute
	// mailOutboxQueueSize bounds the jobs waiting in memory for a sender. An entry past it stays due,
	// and the next poll reads it back.
	mailOutboxQueueSize = 256
	// mailOutboxReleaseTimeout is the end of the drain that releases the entries a shutdown cuts.
	mailOutboxReleaseTimeout = time.Second
)

// mailOutboxRetries holds the delays before each retry of a transient failure, as RFC 5321
// §4.5.4.1 suggests. The last one repeats until the entry expires.
var mailOutboxRetries = []time.Duration{30 * time.Minute, time.Hour, 2 * time.Hour}

var (
	// ErrMailOutboxExpired is returned when an entry's first code expires before a send delivers it.
	ErrMailOutboxExpired = errors.New("mail expired before delivery")
	// ErrMailOutboxUnknownUsage is returned for an entry whose usage has no mail.
	ErrMailOutboxUnknownUsage = errors.New("no mail for short code usage")
)

// MailOutboxDaoList reads the entries due for an attempt.
type MailOutboxDaoList interface {
	Exec(ctx context.Context, request *dao.MailOutboxListRequest) ([]*dao.MailOutboxEntry, error)
}

// MailOutboxDaoClaim takes a due entry for one attempt.
type MailOutboxDaoClaim interface {
	Exec(ctx context.Context, request *dao.MailOutboxClaimRequest) (*dao.MailOutboxEntry, error)
}

// MailOutboxDaoRenew replaces the code of an entry's short code.
type MailOutboxDaoRenew interface {
	Exec(ctx context.Context, request *dao.ShortCodeRenewRequest) (*dao.ShortCode, error)
}

// MailOutboxDaoReschedule sets when an entry's next attempt may start.
type MailOutboxDaoReschedule interface {
	Exec(ctx context.Context, request *dao.MailOutboxRescheduleRequest) (*dao.MailOutboxEntry, error)
}

// MailOutboxDaoDelete ends an entry.
type MailOutboxDaoDelete interface {
	Exec(ctx context.Context, request *dao.MailOutboxDeleteRequest) (*dao.MailOutboxEntry, error)
}

// MailOutboxSMTP sends one rendered mail. Its error matches smtp.ErrTransient when a later attempt
// may deliver the mail.
type MailOutboxSMTP interface {
	SendMail(to smtp.MailUsers, t *template.Template, tName string, data any) error
}

// MailOutboxReport receives each failure that ends an entry, with the entry's usage, under the
// attempt's span. A mail outlives the request that stored it, so its failure surfaces here alone.
type MailOutboxReport func(ctx context.Context, usage string, err error)

// MailOutboxJob names the entry of one short code for an attempt. PlainCode is the code the request
// just issued. An entry read back from the outbox has none, so its attempt renews the code.
type MailOutboxJob struct {
	ShortCodeID uuid.UUID
	PlainCode   string
}

// MailOutbox delivers the mail of each short code from the outbox. A transient failure retries on
// RFC 5321's schedule until the entry's first code expires, and any other outcome ends the entry.
type MailOutbox struct {
	daoList       MailOutboxDaoList
	daoClaim      MailOutboxDaoClaim
	daoRenew      MailOutboxDaoRenew
	daoReschedule MailOutboxDaoReschedule
	daoDelete     MailOutboxDaoDelete
	smtp          MailOutboxSMTP
	limit         int
	report        MailOutboxReport

	shortCodesConfig authconfig.ShortCodes
	smtpConfig       authconfig.SmtpUrls

	jobs chan *MailOutboxJob
	// inflight holds the short code IDs of the entries this instance is attempting.
	inflight sync.Map
}

// NewMailOutbox creates an outbox that sends at most limit mails at once. A limit below one becomes
// one. A nil report leaves an ending failure on its span, for a caller that waits for the result.
func NewMailOutbox(
	daoList MailOutboxDaoList,
	daoClaim MailOutboxDaoClaim,
	daoRenew MailOutboxDaoRenew,
	daoReschedule MailOutboxDaoReschedule,
	daoDelete MailOutboxDaoDelete,
	smtp MailOutboxSMTP,
	limit int,
	report MailOutboxReport,
	shortCodesConfig authconfig.ShortCodes,
	smtpConfig authconfig.SmtpUrls,
) *MailOutbox {
	return &MailOutbox{
		daoList:          daoList,
		daoClaim:         daoClaim,
		daoRenew:         daoRenew,
		daoReschedule:    daoReschedule,
		daoDelete:        daoDelete,
		smtp:             smtp,
		limit:            max(limit, 1),
		report:           report,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
		jobs:             make(chan *MailOutboxJob, mailOutboxQueueSize),
	}
}

// Enqueue hands job to a sender of [MailOutbox.Run] without waiting. A full queue leaves the entry
// due in the outbox, where the next poll finds it.
func (outbox *MailOutbox) Enqueue(_ context.Context, job *MailOutboxJob) {
	select {
	case outbox.jobs <- job:
	default:
	}
}

// Run delivers the outbox until ctx ends. Its senders take the queued jobs, and it queues the due
// entries at once, then every poll interval, so a starting instance resumes the pending mail. Once
// ctx ends, it returns within drain: it waits for the attempts still running, then releases the
// entries of the others, so the next instance retries them without waiting for their lease.
//
// Run lasts the process, so its polls and attempts open their own spans instead.
//
// nosemgrep: a-novel-must-open-span
func (outbox *MailOutbox) Run(ctx context.Context, drain time.Duration) error {
	// An attempt settles its entry after a send, which may finish after ctx ends.
	attemptCtx := context.WithoutCancel(ctx)

	var senders sync.WaitGroup

	for range outbox.limit {
		senders.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-outbox.jobs:
					// Deliver reports the failures that end an entry, and the next attempt follows a
					// transient one.
					_ = outbox.Deliver(attemptCtx, job)
				}
			}
		})
	}

	ticker := time.NewTicker(mailOutboxPollInterval)
	defer ticker.Stop()

	for ctx.Err() == nil {
		outbox.poll(ctx)

		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}

	drained := make(chan struct{})

	go func() {
		senders.Wait()
		close(drained)
	}()

	select {
	case <-drained:
		return nil
	case <-time.After(max(drain-mailOutboxReleaseTimeout, 0)):
		return outbox.release(attemptCtx)
	}
}

// Deliver runs one attempt of the job's entry. A transient failure schedules the next attempt when
// one fits before the entry expires, and any other outcome ends the entry. It returns the attempt's
// failure, and nil when another attempt holds the entry.
func (outbox *MailOutbox) Deliver(ctx context.Context, job *MailOutboxJob) error {
	ctx, span := otel.Tracer().Start(ctx, "core.MailOutbox(deliver)")
	defer span.End()
	defer otel.RecoverPanic(ctx, span)

	span.SetAttributes(attribute.String("short_code.id", job.ShortCodeID.String()))

	now := time.Now()
	claimRequest := &dao.MailOutboxClaimRequest{
		ShortCodeID: job.ShortCodeID,
		Now:         now,
		LeaseUntil:  now.Add(mailOutboxLease),
	}

	entry, err := outbox.daoClaim.Exec(ctx, claimRequest)
	if errors.Is(err, dao.ErrMailOutboxClaimNotFound) {
		// Another attempt holds the entry, or the entry has ended.
		return nil
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("claim entry: %w", err))
	}

	span.SetAttributes(attribute.String("mail.kind", entry.Usage), attribute.Int("mail.attempt", entry.Attempts))

	outbox.inflight.Store(entry.ShortCodeID, struct{}{})
	defer outbox.inflight.Delete(entry.ShortCodeID)

	sendErr := outbox.send(ctx, entry, job.PlainCode, now)
	if errors.Is(sendErr, dao.ErrShortCodeRenewNotFound) {
		// The code was consumed or superseded, so the mail has nothing left to carry.
		sendErr = nil
	}

	retryAt := time.Now().Add(mailOutboxRetries[min(entry.Attempts, len(mailOutboxRetries))-1])

	switch {
	case errors.Is(sendErr, smtp.ErrTransient) && retryAt.Before(entry.ExpiresAt):
		rescheduleRequest := &dao.MailOutboxRescheduleRequest{ShortCodeID: entry.ShortCodeID, DueAt: retryAt}

		_, err = outbox.daoReschedule.Exec(ctx, rescheduleRequest)
		if errors.Is(err, dao.ErrMailOutboxRescheduleNotFound) {
			// An attempt after this one's lease already ended the entry.
			err = nil
		}
	case errors.Is(sendErr, smtp.ErrTransient):
		sendErr = errors.Join(sendErr, ErrMailOutboxExpired)
		err = outbox.finish(ctx, entry, sendErr)
	default:
		err = outbox.finish(ctx, entry, sendErr)
	}

	err = errors.Join(sendErr, err)
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}

// send delivers the entry's mail with plainCode or, without it, with a renewed code.
func (outbox *MailOutbox) send(ctx context.Context, entry *dao.MailOutboxEntry, plainCode string, now time.Time) error {
	ctx, span := otel.Tracer().Start(ctx, "core.MailOutbox(send)")
	defer span.End()

	if !now.Before(entry.ExpiresAt) {
		return otel.ReportError(span, ErrMailOutboxExpired)
	}

	ttl := outbox.shortCodesConfig.Usages[entry.Usage].TTL

	if plainCode == "" {
		code, hash, err := newShortCodeSecret(outbox.shortCodesConfig.Size)
		if err != nil {
			return otel.ReportError(span, err)
		}

		renewRequest := &dao.ShortCodeRenewRequest{ID: entry.ShortCodeID, Code: hash, Now: now, ExpiresAt: now.Add(ttl)}

		_, err = outbox.daoRenew.Exec(ctx, renewRequest)
		if err != nil {
			return otel.ReportError(span, fmt.Errorf("renew short code: %w", err))
		}

		plainCode = code
	}

	mail, data, err := outbox.compose(entry, plainCode, ttl)
	if err != nil {
		return otel.ReportError(span, err)
	}

	err = outbox.smtp.SendMail(smtp.MailUsers{{Email: entry.Recipient}}, mail, entry.Lang, data)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("send mail: %w", err))
	}

	return nil
}

// finish ends the entry. A failure that ends it reaches the report.
func (outbox *MailOutbox) finish(ctx context.Context, entry *dao.MailOutboxEntry, sendErr error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.MailOutbox(finish)")
	defer span.End()

	if sendErr != nil && outbox.report != nil {
		outbox.report(ctx, entry.Usage, sendErr)
	}

	_, err := outbox.daoDelete.Exec(ctx, &dao.MailOutboxDeleteRequest{ShortCodeID: entry.ShortCodeID})
	if err != nil && !errors.Is(err, dao.ErrMailOutboxDeleteNotFound) {
		return otel.ReportError(span, fmt.Errorf("delete entry: %w", err))
	}

	return nil
}

// compose returns the template and the data of the entry's mail.
func (outbox *MailOutbox) compose(
	entry *dao.MailOutboxEntry, plainCode string, ttl time.Duration,
) (*template.Template, map[string]any, error) {
	data := map[string]any{
		mails.TemplateVarShortCode: plainCode,
		mails.TemplateVarTarget:    entry.Target,
		mails.TemplateVarDuration:  ttl.Hours(),
		mails.TemplateVarBanner:    assets.BannerBase64,
	}

	switch entry.Usage {
	case ShortCodeUsageRegister:
		data[mails.TemplateVarTarget] = base64.RawURLEncoding.EncodeToString([]byte(entry.Target))
		data[mails.TemplateVarURL] = outbox.smtpConfig.Register

		return mails.Mails.Register, data, nil
	case ShortCodeUsageValidateEmail:
		data[mails.TemplateVarSource] = base64.RawURLEncoding.EncodeToString([]byte(entry.Recipient))
		data[mails.TemplateVarURL] = outbox.smtpConfig.UpdateEmail

		return mails.Mails.EmailUpdate, data, nil
	case ShortCodeUsageResetPassword:
		data[mails.TemplateVarURL] = outbox.smtpConfig.UpdatePassword

		return mails.Mails.PasswordReset, data, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s", ErrMailOutboxUnknownUsage, entry.Usage)
	}
}

// poll queues the due entries the queue has room for. A failed read waits for the next poll.
func (outbox *MailOutbox) poll(ctx context.Context) {
	ctx, span := otel.Tracer().Start(ctx, "core.MailOutbox(poll)")
	defer span.End()

	room := cap(outbox.jobs) - len(outbox.jobs)
	if room == 0 {
		return
	}

	entries, err := outbox.daoList.Exec(ctx, &dao.MailOutboxListRequest{Now: time.Now(), Limit: room})
	if err != nil {
		_ = otel.ReportError(span, fmt.Errorf("list due entries: %w", err))

		return
	}

	for _, entry := range entries {
		outbox.Enqueue(ctx, &MailOutboxJob{ShortCodeID: entry.ShortCodeID})
	}
}

// release makes the entries of the attempts still running due at once, so the next instance retries
// them instead of waiting for their lease. An attempt cut after the final "." may then deliver twice.
func (outbox *MailOutbox) release(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "core.MailOutbox(release)")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, mailOutboxReleaseTimeout)
	defer cancel()

	errs := []error{context.DeadlineExceeded}
	now := time.Now()

	outbox.inflight.Range(func(key, _ any) bool {
		shortCodeID, _ := key.(uuid.UUID)

		_, err := outbox.daoReschedule.Exec(ctx, &dao.MailOutboxRescheduleRequest{ShortCodeID: shortCodeID, DueAt: now})
		if err != nil && !errors.Is(err, dao.ErrMailOutboxRescheduleNotFound) {
			errs = append(errs, err)
		}

		return true
	})

	return otel.ReportError(span, fmt.Errorf("drain mail outbox: %w", errors.Join(errs...)))
}
