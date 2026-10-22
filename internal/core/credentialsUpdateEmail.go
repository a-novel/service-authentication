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
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// CredentialsUpdateEmailDao assigns the new email to the account.
type CredentialsUpdateEmailDao interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateEmailRequest) (*dao.Credentials, error)
}

// CredentialsUpdateEmailServiceShortCodeVerify checks the email-change short code.
type CredentialsUpdateEmailServiceShortCodeVerify interface {
	Exec(ctx context.Context, request *ShortCodeVerifyRequest) (*ShortCode, error)
}

// CredentialsUpdateEmailServiceShortCodeConsume retires the verified short code.
type CredentialsUpdateEmailServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

// CredentialsUpdateEmailRequest names the account and the short code confirming its new email.
type CredentialsUpdateEmailRequest struct {
	UserID    uuid.UUID
	ShortCode string `validate:"required,max=1024"`
}

// CredentialsUpdateEmail applies an email change confirmed by a short code. The
// caller does not supply the new address directly: it is carried in the short-code
// payload, so only the address the code was issued for can take effect.
type CredentialsUpdateEmail struct {
	dao                     CredentialsUpdateEmailDao
	serviceShortCodeVerify  CredentialsUpdateEmailServiceShortCodeVerify
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume
	transactor              transaction.Transactor
}

func NewCredentialsUpdateEmail(
	dao CredentialsUpdateEmailDao,
	serviceShortCodeVerify CredentialsUpdateEmailServiceShortCodeVerify,
	serviceShortCodeConsume CredentialsUpdateEmailServiceShortCodeConsume,
	transactor transaction.Transactor,
) *CredentialsUpdateEmail {
	return &CredentialsUpdateEmail{
		dao:                     dao,
		serviceShortCodeVerify:  serviceShortCodeVerify,
		serviceShortCodeConsume: serviceShortCodeConsume,
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

	var credentials *dao.Credentials

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{ID: shortCode.ID})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		daoRequest := &dao.CredentialsUpdateEmailRequest{
			ID:    request.UserID,
			Email: newEmail,
			Now:   time.Now(),
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

	return newCredentials(credentials), nil
}
