package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

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

// ShortCodeCreateRegisterMailDelivery reserves bounded delivery capacity for the registration mail.
type ShortCodeCreateRegisterMailDelivery interface {
	Reserve(ctx context.Context) (MailDeliveryReservation, error)
}

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
	mailDelivery     ShortCodeCreateRegisterMailDelivery
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls
}

// NewShortCodeCreateRegister wires the registration flow to the short-code
// service, the email-existence DAO, and the delivery owner.
func NewShortCodeCreateRegister(
	service ShortCodeCreateRegisterService,
	selectDao ShortCodeCreateRegisterDao,
	mailDelivery ShortCodeCreateRegisterMailDelivery,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{
		service:          service,
		selectDao:        selectDao,
		mailDelivery:     mailDelivery,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
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

	reservation, err := service.mailDelivery.Reserve(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("reserve mail delivery: %w", err))
	}
	defer reservation.Release()

	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageRegister,
		Target:   request.Email,
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	reservation.Deliver(ctx, &MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: request.Email}},
		Template:     mails.Mails.Register,
		TemplateName: request.Lang,
		Data: map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
			mails.TemplateVarURL:       service.smtpConfig.Register,
			mails.TemplateVarDuration:  service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL.Hours(),
			mails.TemplateVarBanner:    assets.BannerBase64,
			mails.TemplateVarPurpose:   mailDeliveryKindRegister,
		},
		Kind: mailDeliveryKindRegister,
	})

	return otel.ReportSuccess(span, shortCode), nil
}
