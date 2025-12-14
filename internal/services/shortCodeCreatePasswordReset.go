package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

type ShortCodeCreatePasswordResetService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}
type ShortCodeCreatePasswordResetRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}
type ShortCodeCreatePasswordResetSmtp = smtp.Sender

type ShortCodeCreatePasswordResetRequest struct {
	Email string `validate:"required,email,max=1024"`
	Lang  string `validate:"required,langs"`
}

type ShortCodeCreatePasswordReset struct {
	service          ShortCodeCreatePasswordResetService
	selectRepository ShortCodeCreatePasswordResetRepository
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

func NewShortCodeCreatePasswordReset(
	service ShortCodeCreatePasswordResetService,
	selectRepository ShortCodeCreatePasswordResetRepository,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreatePasswordReset {
	return &ShortCodeCreatePasswordReset{
		service:          service,
		selectRepository: selectRepository,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

func (service *ShortCodeCreatePasswordReset) Wait() {
	service.wg.Wait()
}

func (service *ShortCodeCreatePasswordReset) Exec(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreatePasswordReset")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	credentials, err := service.selectRepository.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	// Create a new short code.
	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(ctx), request, credentials.ID, shortCode)

	return otel.ReportSuccess(span, shortCode), nil
}

func (service *ShortCodeCreatePasswordReset) sendMail(
	ctx context.Context, request *ShortCodeCreatePasswordResetRequest, userID uuid.UUID, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.RequestPasswordReset.sendMail")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.PasswordReset,
		request.Lang,
		map[string]any{
			"ShortCode": shortCode.PlainCode,
			"Target":    userID.String(),
			"URL":       service.smtpConfig.UpdatePassword,
			"Duration":  service.shortCodesConfig.Usages[ShortCodeUsageResetPassword].TTL.Hours(),
			"Banner":    assets.BannerBase64,
			"_Purpose":  "password-reset",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "password reset request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
