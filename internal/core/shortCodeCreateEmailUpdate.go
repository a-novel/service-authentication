package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

// ShortCodeCreateEmailUpdateService issues the underlying short code; satisfied
// by [ShortCodeCreate].
type ShortCodeCreateEmailUpdateService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}

// ShortCodeCreateEmailUpdateDao reports whether an email address is already
// registered, so no code is mailed to an address already in use.
type ShortCodeCreateEmailUpdateDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// ShortCodeCreateEmailUpdateMailDelivery reserves bounded capacity for the confirmation mail.
type ShortCodeCreateEmailUpdateMailDelivery interface {
	Reserve(ctx context.Context) (MailDeliveryReservation, error)
}

// ShortCodeCreateEmailUpdateRequest carries the user changing their address, the
// new email to confirm, and the language of the confirmation mail.
type ShortCodeCreateEmailUpdateRequest struct {
	ID    uuid.UUID
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

// ShortCodeCreateEmailUpdate issues a [ShortCodeUsageValidateEmail] code for an
// email-change confirmation and emails it to the prospective new address, unless
// that address is already registered.
type ShortCodeCreateEmailUpdate struct {
	service          ShortCodeCreateEmailUpdateService
	selectDao        ShortCodeCreateEmailUpdateDao
	mailDelivery     ShortCodeCreateEmailUpdateMailDelivery
	shortCodesConfig authconfig.ShortCodes
	smtpConfig       authconfig.SmtpUrls
}

// NewShortCodeCreateEmailUpdate wires the email-change flow to the short-code
// service, the email-existence DAO, and the delivery owner.
func NewShortCodeCreateEmailUpdate(
	service ShortCodeCreateEmailUpdateService,
	selectDao ShortCodeCreateEmailUpdateDao,
	mailDelivery ShortCodeCreateEmailUpdateMailDelivery,
	shortCodesConfig authconfig.ShortCodes,
	smtpConfig authconfig.SmtpUrls,
) *ShortCodeCreateEmailUpdate {
	return &ShortCodeCreateEmailUpdate{
		service:          service,
		selectDao:        selectDao,
		mailDelivery:     mailDelivery,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

// Exec issues the confirmation code and schedules its delivery email, returning
// the code immediately.
//
// A registered address gets the same code and result but no mail, so neither
// this call nor the pending change it leaves reveals that the address is taken.
// Without the mailed code the change cannot be confirmed, and
// [CredentialsUpdateEmail] would reject the taken address anyway.
func (service *ShortCodeCreateEmailUpdate) Exec(
	ctx context.Context, request *ShortCodeCreateEmailUpdateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCreateEmailUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", request.ID.String()),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	_, err = service.selectDao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{Email: request.Email})
	emailTaken := err == nil

	if !emailTaken && !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("check existing email: %w", err))
	}

	// A taken address reserves too, so a saturated mailer fails both cases alike. The deferred
	// Release returns the slot when no mail goes out.
	reservation, err := service.mailDelivery.Reserve(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("reserve mail delivery: %w", err))
	}
	defer reservation.Release()

	ttl := service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL
	createRequest := &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageValidateEmail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      ttl,
		Override: true,
	}

	shortCode, err := service.service.Exec(ctx, createRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	if emailTaken {
		return shortCode, nil
	}

	mail := &MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: request.Email}},
		Template:     mails.Mails.EmailUpdate,
		TemplateName: request.Lang,
		Data: map[string]any{
			mails.TemplateVarShortCode: shortCode.PlainCode,
			mails.TemplateVarTarget:    request.ID.String(),
			mails.TemplateVarSource:    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
			mails.TemplateVarURL:       service.smtpConfig.UpdateEmail,
			mails.TemplateVarDuration:  ttl.Hours(),
		},
		Kind: mailDeliveryKindEmailUpdate,
	}

	reservation.Deliver(ctx, mail)

	return shortCode, nil
}
