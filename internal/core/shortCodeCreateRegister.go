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

// ShortCodeCreateRegisterRequest carries the address and role to register and the
// language of the registration mail.
type ShortCodeCreateRegisterRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
	Role  string `validate:"required,role"`
}

// ShortCodeCreateRegister issues a [ShortCodeUsageRegister] code for a new-account
// sign-up and emails it to the prospective address, after rejecting the sign-up if
// that address is already registered.
type ShortCodeCreateRegister struct {
	service          ShortCodeCreateRegisterService
	selectDao        ShortCodeCreateRegisterDao
	shortCodesConfig authconfig.ShortCodes
}

// NewShortCodeCreateRegister wires the registration flow to the short-code
// service, the email-existence DAO.
func NewShortCodeCreateRegister(
	service ShortCodeCreateRegisterService,
	selectDao ShortCodeCreateRegisterDao,
	shortCodesConfig authconfig.ShortCodes,
) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{
		service:          service,
		selectDao:        selectDao,
		shortCodesConfig: shortCodesConfig,
	}
}

// Exec issues the registration code and schedules its delivery email, returning
// the code immediately. It fails if the address is already registered.
func (service *ShortCodeCreateRegister) Exec(
	ctx context.Context, request *ShortCodeCreateRegisterRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCreateRegister")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.email", request.Email),
		attribute.String("email.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	_, err = service.selectDao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{Email: request.Email})
	if err == nil {
		return nil, otel.ReportError(span, dao.ErrCredentialsInsertAlreadyExists)
	}

	if !errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
		return nil, otel.ReportError(span, fmt.Errorf("check existing email: %w", err))
	}

	ttl := service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL
	createRequest := &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageRegister,
		Target:   request.Email,
		Data:     credentialsCreateRegistrationData{Role: request.Role},
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
