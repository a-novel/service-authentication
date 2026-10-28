package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
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
	shortCodesConfig authconfig.ShortCodes
}

// NewShortCodeCreatePasswordReset wires the password-reset flow to the short-code
// service, the credentials lookup DAO.
func NewShortCodeCreatePasswordReset(
	service ShortCodeCreatePasswordResetService,
	selectDao ShortCodeCreatePasswordResetDao,
	shortCodesConfig authconfig.ShortCodes,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{
		service:          service,
		selectDao:        selectDao,
		shortCodesConfig: shortCodesConfig,
	}
}

// Exec issues the reset code and schedules its delivery email, returning the code
// immediately. It fails if no account matches the requested email.
func (service *ShortCodeCreatePasswordReset) Exec(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCreatePasswordReset")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	credentials, err := service.selectDao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{Email: request.Email})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	ttl := service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL
	createRequest := &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      ttl,
		Override: true,
		Mail:     &ShortCodeCreateMail{Recipient: credentials.Email, Lang: request.Lang},
	}

	shortCode, err := service.service.Exec(ctx, createRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	return shortCode, nil
}
