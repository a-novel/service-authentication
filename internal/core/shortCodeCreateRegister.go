package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

// ShortCodeCreateRegisterService issues the underlying short code; satisfied by
// [ShortCodeCreate].
type ShortCodeCreateRegisterService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}

// ShortCodeCreateRegisterDao reports whether an email address is already
// registered, so a duplicate sign-up can be rejected.
type ShortCodeCreateRegisterDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// ShortCodeCreateRegisterSmtp is the mailer used to deliver the registration code.
type ShortCodeCreateRegisterSmtp = smtp.Sender

// ShortCodeCreateRegisterRequest carries the address to register and the language
// of the registration mail.
type ShortCodeCreateRegisterRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

// ShortCodeCreateRegister issues a [ShortCodeUsageRegister] code for a new-account
// sign-up and emails it to the prospective address, after rejecting the sign-up if
// that address is already registered.
type ShortCodeCreateRegister struct {
	service          ShortCodeCreateRegisterService
	selectDao        ShortCodeCreateRegisterDao
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

// NewShortCodeCreateRegister wires the registration flow to the short-code
// service, the email-existence DAO, and the mailer.
func NewShortCodeCreateRegister(
	service ShortCodeCreateRegisterService,
	selectDao ShortCodeCreateRegisterDao,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{
		service:          service,
		selectDao:        selectDao,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

// Wait blocks until every in-flight registration email has finished sending, so
// callers can drain pending deliveries before shutdown.
func (service *ShortCodeCreateRegister) Wait() {
	service.wg.Wait()
}

// Exec issues the registration code and schedules its delivery email, returning
// the code immediately. It fails if the address is already registered.
func (service *ShortCodeCreateRegister) Exec(
	ctx context.Context, request *ShortCodeCreateRegisterRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateRegister")
	defer span.End()

	span.SetAttributes(
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
		return nil, dao.ErrCredentialsInsertAlreadyExists
	}

	if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("check existing email: %w", err))
	}

	// Create a new short code.
	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageRegister,
		Target:   request.Email,
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL,
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

func (service *ShortCodeCreateRegister) sendMail(
	ctx context.Context, request *ShortCodeCreateRegisterRequest, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateRegister(sendMail)")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.Register,
		request.Lang,
		map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
			mails.TemplateVarURL:       service.smtpConfig.Register,
			mails.TemplateVarDuration:  service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL.Hours(),
			mails.TemplateVarBanner:    assets.BannerBase64,
			mails.TemplateVarPurpose:   "register",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "register request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
