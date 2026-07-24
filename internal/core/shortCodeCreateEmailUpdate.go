package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

// ShortCodeCreateEmailUpdateService issues the underlying short code; satisfied
// by [ShortCodeCreate].
type ShortCodeCreateEmailUpdateService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}

// ShortCodeCreateEmailUpdateDao reports whether an email address is already
// registered, so a change to an address already in use can be rejected.
type ShortCodeCreateEmailUpdateDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// ShortCodeCreateEmailUpdateSmtp is the mailer used to deliver the confirmation code.
type ShortCodeCreateEmailUpdateSmtp = smtp.Sender

// ShortCodeCreateEmailUpdateRequest carries the user changing their address, the
// new email to confirm, and the language of the confirmation mail.
type ShortCodeCreateEmailUpdateRequest struct {
	ID    uuid.UUID
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

// ShortCodeCreateEmailUpdate issues a [ShortCodeUsageValidateEmail] code for an
// email-change confirmation and emails it to the prospective new address, after
// rejecting the change if that address is already registered.
type ShortCodeCreateEmailUpdate struct {
	service          ShortCodeCreateEmailUpdateService
	selectDao        ShortCodeCreateEmailUpdateDao
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

// NewShortCodeCreateEmailUpdate wires the email-change flow to the short-code
// service, the email-existence DAO, and the mailer.
func NewShortCodeCreateEmailUpdate(
	service ShortCodeCreateEmailUpdateService,
	selectDao ShortCodeCreateEmailUpdateDao,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreateEmailUpdate {
	return &ShortCodeCreateEmailUpdate{
		service:          service,
		selectDao:        selectDao,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

// Wait blocks until every in-flight confirmation email has finished sending, so
// callers can drain pending deliveries before shutdown.
func (service *ShortCodeCreateEmailUpdate) Wait() {
	service.wg.Wait()
}

// Exec issues the confirmation code and schedules its delivery email, returning
// the code immediately. It fails if the requested email is already registered.
func (service *ShortCodeCreateEmailUpdate) Exec(
	ctx context.Context, request *ShortCodeCreateEmailUpdateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateEmailUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.ID.String()),
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	_, err = service.selectDao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err == nil {
		return nil, dao.ErrCredentialsUpdateEmailAlreadyExists
	}

	if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("check existing email: %w", err))
	}

	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageValidateEmail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	// Deliver the code by email in a detached goroutine: WithoutCancel keeps the
	// send alive after the request context is cancelled, and Wait drains it on shutdown.
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(ctx), request, shortCode)

	return otel.ReportSuccess(span, shortCode), nil
}

func (service *ShortCodeCreateEmailUpdate) sendMail(
	ctx context.Context, request *ShortCodeCreateEmailUpdateRequest, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateEmailUpdate(sendMail)")
	defer span.End()
	defer otel.RecoverPanic(ctx, span)

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.EmailUpdate,
		request.Lang,
		map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    request.ID.String(),
			"Source":                   base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
			mails.TemplateVarURL:       service.smtpConfig.UpdateEmail,
			mails.TemplateVarDuration:  service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL.Hours(),
			mails.TemplateVarBanner:    assets.BannerBase64,
			mails.TemplateVarPurpose:   "email-update",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "email update request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
