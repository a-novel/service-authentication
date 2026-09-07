package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
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

// ShortCodeCreatePasswordResetMailDelivery reserves bounded capacity for the reset mail.
type ShortCodeCreatePasswordResetMailDelivery interface {
	Reserve(ctx context.Context) (MailDeliveryReservation, error)
}

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
	mailDelivery     ShortCodeCreatePasswordResetMailDelivery
	shortCodesConfig authconfig.ShortCodes
	smtpConfig       authconfig.SmtpUrls
}

// NewShortCodeCreatePasswordReset wires the password-reset flow to the short-code
// service, the credentials lookup DAO, and the delivery owner.
func NewShortCodeCreatePasswordReset(
	service ShortCodeCreatePasswordResetService,
	selectDao ShortCodeCreatePasswordResetDao,
	mailDelivery ShortCodeCreatePasswordResetMailDelivery,
	shortCodesConfig authconfig.ShortCodes,
	smtpConfig authconfig.SmtpUrls,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{
		service:          service,
		selectDao:        selectDao,
		mailDelivery:     mailDelivery,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
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

	reservation, err := service.mailDelivery.Reserve(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("reserve mail delivery: %w", err))
	}
	defer reservation.Release()

	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	reservation.Deliver(ctx, &MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: request.Email}},
		Template:     mails.Mails.PasswordReset,
		TemplateName: request.Lang,
		Data: map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    credentials.ID.String(),
			mails.TemplateVarURL:       service.smtpConfig.UpdatePassword,
			mails.TemplateVarDuration:  service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL.Hours(),
			mails.TemplateVarBanner:    assets.BannerBase64,
			mails.TemplateVarPurpose:   mailDeliveryKindPasswordReset,
		},
		Kind: mailDeliveryKindPasswordReset,
	})

	return otel.ReportSuccess(span, shortCode), nil
}
