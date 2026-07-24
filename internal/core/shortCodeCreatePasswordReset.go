package core

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

// ShortCodeCreatePasswordResetService issues the underlying short code; satisfied
// by [ShortCodeCreate].
type ShortCodeCreatePasswordResetService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}

// ShortCodeCreatePasswordResetDao looks up the credentials for an email address,
// which both confirms the account exists and yields the user ID to bind the code to.
type ShortCodeCreatePasswordResetDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// ShortCodeCreatePasswordResetSmtp is the mailer used to deliver the reset code.
type ShortCodeCreatePasswordResetSmtp = smtp.Sender

// ShortCodeCreatePasswordResetRequest carries the account email to reset and the
// language of the reset mail.
type ShortCodeCreatePasswordResetRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

// ShortCodeCreatePasswordReset issues a [ShortCodeUsageResetPassword] code for an
// existing account and emails it to the account's current address.
type ShortCodeCreatePasswordReset struct {
	service          ShortCodeCreatePasswordResetService
	selectDao        ShortCodeCreatePasswordResetDao
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

// NewShortCodeCreatePasswordReset wires the password-reset flow to the short-code
// service, the credentials lookup DAO, and the mailer.
func NewShortCodeCreatePasswordReset(
	service ShortCodeCreatePasswordResetService,
	selectDao ShortCodeCreatePasswordResetDao,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{
		service:          service,
		selectDao:        selectDao,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

// Wait blocks until every in-flight reset email has finished sending, so callers
// can drain pending deliveries before shutdown.
func (service *ShortCodeCreatePasswordReset) Wait() {
	service.wg.Wait()
}

// Exec issues the reset code and schedules its delivery email, returning the code
// immediately. It fails if no account matches the requested email.
func (service *ShortCodeCreatePasswordReset) Exec(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreatePasswordReset")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	credentials, err := service.selectDao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	// Deliver the code by email in a detached goroutine: WithoutCancel keeps the
	// send alive after the request context is cancelled, and Wait drains it on shutdown.
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(ctx), request, credentials.ID, shortCode)

	return otel.ReportSuccess(span, shortCode), nil
}

func (service *ShortCodeCreatePasswordReset) sendMail(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest, userID uuid.UUID, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.RequestPasswordReset.sendMail")
	defer span.End()
	defer lib.RecoverPanic(ctx, span)

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.PasswordReset,
		request.Lang,
		map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    userID.String(),
			mails.TemplateVarURL:       service.smtpConfig.UpdatePassword,
			mails.TemplateVarDuration:  service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL.Hours(),
			mails.TemplateVarBanner:    assets.BannerBase64,
			mails.TemplateVarPurpose:   "password-reset",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "password reset request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
