package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
)

// RequestEmailUpdateSource is the source used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
type RequestEmailUpdateSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	smtp.Sender
}

func NewRequestEmailUpdateServiceSource(
	createShortCode *CreateShortCodeService,
	smtpSender smtp.Sender,
) RequestEmailUpdateSource {
	return &struct {
		*CreateShortCodeService
		smtp.Sender
	}{
		CreateShortCodeService: createShortCode,
		Sender:                 smtpSender,
	}
}

// RequestEmailUpdateRequest is the input used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
type RequestEmailUpdateRequest struct {
	// ID of the account trying to update its email.
	ID uuid.UUID
	// New email of the account. This email will receive a link to confirm the email update.
	Email string
	// Language of the account.
	Lang models.Lang
}

// RequestEmailUpdateService is the service used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
//
// You may create one using the NewRequestEmailUpdateService function.
type RequestEmailUpdateService struct {
	source           RequestEmailUpdateSource
	shortCodesConfig config.ShortCodes
	smtpConfig       models.SMTPURLsConfig
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func NewRequestEmailUpdateService(
	source RequestEmailUpdateSource,
	shortCodesConfig config.ShortCodes,
	smtpConfig models.SMTPURLsConfig,
) *RequestEmailUpdateService {
	return &RequestEmailUpdateService{source: source, shortCodesConfig: shortCodesConfig, smtpConfig: smtpConfig}
}

func (service *RequestEmailUpdateService) Wait() {
	service.wg.Wait()
}

// RequestEmailUpdate requests an email update for a specific account.
//
// The link to update the email is sent to the new email address provided in the request. This also serves as
// a validation for the new email address.
func (service *RequestEmailUpdateService) RequestEmailUpdate(
	ctx context.Context, request RequestEmailUpdateRequest,
) (*models.ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.RequestEmailUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.id", request.ID.String()),
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang.String()),
	)

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(ctx, CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageValidateMail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      service.shortCodesConfig.Usages[models.ShortCodeUsageValidateMail].TTL,
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

func (service *RequestEmailUpdateService) sendMail(
	ctx context.Context, request RequestEmailUpdateRequest, shortCode *models.ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.RequestEmailUpdate.sendMail")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang.String()),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.source.SendMail(
		[]string{request.Email},
		models.Mails.EmailUpdate,
		request.Lang.String(),
		map[string]any{
			"ShortCode": shortCode.PlainCode,
			"Target":    request.ID.String(),
			"URL":       service.smtpConfig.UpdateEmail,
			"Duration":  service.shortCodesConfig.Usages[models.ShortCodeUsageValidateMail].TTL.String(),
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
