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
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

type ShortCodeCreateEmailUpdateService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}
type ShortCodeCreateEmailUpdateSmtp = smtp.Sender

type ShortCodeCreateEmailUpdateRequest struct {
	ID    uuid.UUID
	Email string `validate:"required,email"`
	Lang  string `validate:"required,langs"`
}

type ShortCodeCreateEmailUpdate struct {
	service          ShortCodeCreateEmailUpdateService
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

func NewShortCodeCreateEmailUpdate(
	service ShortCodeCreateEmailUpdateService,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreateEmailUpdate {
	return &ShortCodeCreateEmailUpdate{
		service:          service,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

func (service *ShortCodeCreateEmailUpdate) Wait() {
	service.wg.Wait()
}

func (service *ShortCodeCreateEmailUpdate) Exec(
	ctx context.Context, request *ShortCodeCreateEmailUpdateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateEmailUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.id", request.ID.String()),
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Create a new short code.
	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageValidateEmail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL,
		Override: true,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(ctx), request, shortCode)

	return otel.ReportSuccess(span, shortCode), nil
}

func (service *ShortCodeCreateEmailUpdate) sendMail(
	ctx context.Context, request *ShortCodeCreateEmailUpdateRequest, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateEmailUpdate(sendMail)")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.EmailUpdate,
		request.Lang,
		map[string]any{
			"ShortCode": shortCode.PlainCode,
			"Target":    request.ID.String(),
			"URL":       service.smtpConfig.UpdateEmail,
			"Duration":  service.shortCodesConfig.Usages[ShortCodeUsageValidateEmail].TTL.Hours(),
			"Banner":    assets.BannerBase64,
			"_Purpose":  "email-update",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "email update request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
