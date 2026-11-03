package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
)

// CredentialsUpdateEmailDao assigns the new email to the account.
type CredentialsUpdateEmailDao interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateEmailRequest) (*dao.Credentials, error)
}

// CredentialsUpdateEmailDaoSelect loads the account before the change, for the address it leaves.
type CredentialsUpdateEmailDaoSelect interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

// CredentialsUpdateEmailServiceShortCodeVerify checks the email-change short code.
type CredentialsUpdateEmailServiceShortCodeVerify interface {
	Exec(ctx context.Context, request *ShortCodeVerifyRequest) (*ShortCode, error)
}

// CredentialsUpdateEmailServiceShortCodeConsume retires the verified short code.
type CredentialsUpdateEmailServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

// CredentialsUpdateEmailMailDelivery reserves bounded capacity for the notice to the former address.
type CredentialsUpdateEmailMailDelivery interface {
	Reserve(ctx context.Context) (MailDeliveryReservation, error)
}

// CredentialsUpdateEmailRequest names the account, the short code confirming its new email, and the
// language of the notice the former address receives.
type CredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID
	ShortCode string `validate:"required,max=1024"`
	Lang      string `validate:"required,langs"`
}

// CredentialsUpdateEmail applies an email change confirmed by a short code. The
// caller does not supply the new address directly: it is carried in the short-code
// payload, so only the address the code was issued for can take effect.
//
// The change ends every session opened before it, whose refresh tokens no longer renew, and mails
// the former address a notice.
type CredentialsUpdateEmail struct {
	dao                     CredentialsUpdateEmailDao
	daoSelect               CredentialsUpdateEmailDaoSelect
	serviceShortCodeVerify  CredentialsUpdateEmailServiceShortCodeVerify
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume
	mailDelivery            CredentialsUpdateEmailMailDelivery
	transactor              transaction.Transactor
}

func NewCredentialsUpdateEmail(
	dao CredentialsUpdateEmailDao,
	daoSelect CredentialsUpdateEmailDaoSelect,
	serviceShortCodeVerify CredentialsUpdateEmailServiceShortCodeVerify,
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume,
	mailDelivery CredentialsUpdateEmailMailDelivery,
	transactor transaction.Transactor,
) *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{
		dao:                     dao,
		daoSelect:               daoSelect,
		serviceShortCodeVerify:  serviceShortCodeVerify,
		serviceShortCodeConsume: serviceShortCodeConsume,
		mailDelivery:            mailDelivery,
		transactor:              transactor,
	}
}

func (service *CredentialsUpdateEmail) Exec(
	ctx context.Context, request *CredentialsUpdateEmailRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsUpdateEmail")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.UserID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// The check runs Argon2, so it happens before the transaction, which holds a pooled connection.
	verifyRequest := &ShortCodeVerifyRequest{
		Usage:  ShortCodeUsageValidateEmail,
		Target: request.UserID.String(),
		Code:   request.ShortCode,
	}

	shortCode, err := service.serviceShortCodeVerify.Exec(ctx, verifyRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("verify short code: %w", err))
	}

	var newEmail string

	err = json.Unmarshal(shortCode.Data, &newEmail)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal short code data: %w", err))
	}

	// The notice is reserved first, so a change never applies without one.
	reservation, err := service.mailDelivery.Reserve(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("reserve mail delivery: %w", err))
	}
	defer reservation.Release()

	var previous, credentials *dao.Credentials

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		var err error

		previous, err = service.daoSelect.Exec(ctx, &dao.CredentialsSelectRequest{ID: request.UserID})
		if err != nil {
			return fmt.Errorf("select credentials: %w", err)
		}

		_, err = service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{ID: shortCode.ID})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		// Whole seconds, as a token's iat: the change ends every session opened before it.
		daoRequest := &dao.CredentialsUpdateEmailRequest{
			ID:    request.UserID,
			Email: newEmail,
			Now:   time.Now().Truncate(time.Second),
		}

		credentials, err = service.dao.Exec(ctx, daoRequest)
		if err != nil {
			return fmt.Errorf("update email: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	reservation.Deliver(ctx, &MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: previous.Email}},
		Template:     mails.Mails.EmailChanged,
		TemplateName: request.Lang,
		Kind:         mailDeliveryKindEmailChanged,
	})

	return newCredentials(credentials), nil
}
