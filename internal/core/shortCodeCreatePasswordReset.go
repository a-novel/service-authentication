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
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

// ShortCodeCreatePasswordResetServiceShortCodeCreate issues the underlying short code; satisfied
// by [ShortCodeCreate].
type ShortCodeCreatePasswordResetServiceShortCodeCreate interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}

// ShortCodeCreatePasswordResetDaoSelectByEmail looks up the credentials for an email address,
// which both confirms the account exists and yields the user ID to bind the code to.
type ShortCodeCreatePasswordResetDaoSelectByEmail interface {
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
	serviceShortCodeCreate ShortCodeCreatePasswordResetServiceShortCodeCreate
	daoSelectByEmail       ShortCodeCreatePasswordResetDaoSelectByEmail
	mailDelivery           ShortCodeCreatePasswordResetMailDelivery
	shortCodesConfig       authconfig.ShortCodes
	smtpConfig             authconfig.SMTPURLs
}

// NewShortCodeCreatePasswordReset wires the password-reset flow to the short-code
// service, the credentials lookup DAO, and the delivery owner.
func NewShortCodeCreatePasswordReset(
	serviceShortCodeCreate ShortCodeCreatePasswordResetServiceShortCodeCreate,
	daoSelectByEmail ShortCodeCreatePasswordResetDaoSelectByEmail,
	mailDelivery ShortCodeCreatePasswordResetMailDelivery,
	shortCodesConfig authconfig.ShortCodes,
	smtpConfig authconfig.SMTPURLs,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{
		serviceShortCodeCreate: serviceShortCodeCreate,
		daoSelectByEmail:       daoSelectByEmail,
		mailDelivery:           mailDelivery,
		shortCodesConfig:       shortCodesConfig,
		smtpConfig:             smtpConfig,
	}
}

// Exec issues the reset code and schedules its delivery email. It fails if no account matches the
// requested email.
//
// An unknown email costs what a known one does: it takes a mail slot, so a saturated mailer fails
// both alike, and runs one Argon2id hash, the bulk of a short code's cost. The response time then
// tells nothing about which addresses are registered.
func (service *ShortCodeCreatePasswordReset) Exec(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCreatePasswordReset")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	reservation, err := service.mailDelivery.Reserve(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("reserve mail delivery: %w", err))
	}
	defer reservation.Release()

	credentials, err := service.daoSelectByEmail.Exec(ctx, &dao.CredentialsSelectByEmailRequest{Email: request.Email})
	if errors.Is(err, dao.ErrCredentialsNotFound) {
		lib.DummyCompareArgon2(request.Email)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	ttl := service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL
	createRequest := &ShortCodeCreateRequest{
		Usage:  ShortCodeUsageResetPassword,
		Target: credentials.ID.String(),
		TTL:    ttl,
	}

	shortCode, err := service.serviceShortCodeCreate.Exec(ctx, createRequest)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	mail := &MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: request.Email}},
		Template:     mails.Mails.PasswordReset,
		TemplateName: request.Lang,
		Data: mails.Link{
			ShortCode: shortCode.PlainCode,
			Target:    credentials.ID.String(),
			URL:       service.smtpConfig.UpdatePassword,
			Duration:  ttl.Hours(),
		},
		Kind: mailDeliveryKindPasswordReset,
	}

	reservation.Deliver(ctx, mail)

	return nil
}
