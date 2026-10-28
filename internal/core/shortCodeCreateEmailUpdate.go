package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
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
	shortCodesConfig authconfig.ShortCodes
}

// NewShortCodeCreateEmailUpdate wires the email-change flow to the short-code
// service, the email-existence DAO.
func NewShortCodeCreateEmailUpdate(
	service ShortCodeCreateEmailUpdateService,
	selectDao ShortCodeCreateEmailUpdateDao,
	shortCodesConfig authconfig.ShortCodes,
) *ShortCodeCreateEmailUpdate {
	return &ShortCodeCreateEmailUpdate{
		service:          service,
		selectDao:        selectDao,
		shortCodesConfig: shortCodesConfig,
	}
}

// Exec issues the confirmation code and schedules its delivery email, returning
// the code immediately. It fails if the requested email is already registered.
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
	if err == nil {
		return nil, otel.ReportError(span, dao.ErrCredentialsUpdateEmailAlreadyExists)
	}

	if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("check existing email: %w", err))
	}

	ttl := service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL
	createRequest := &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageValidateEmail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      ttl,
		Override: true,
		Mail:     &ShortCodeCreateMail{Recipient: request.Email, Lang: request.Lang},
	}

	shortCode, err := service.service.Exec(ctx, createRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	return shortCode, nil
}
