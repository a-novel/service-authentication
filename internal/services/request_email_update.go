package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/a-novel/service-authentication/models"
)

var ErrRequestEmailUpdateService = errors.New("RequestEmailUpdateService.RequestEmailUpdate")

func NewErrRequestEmailUpdateService(err error) error {
	return errors.Join(err, ErrRequestEmailUpdateService)
}

// RequestEmailUpdateSource is the source used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
type RequestEmailUpdateSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	SMTP(ctx context.Context, message *template.Template, lang models.Lang, tos []string, data any)
}

func NewRequestEmailUpdateServiceSource(
	createShortCode *CreateShortCodeService,
	smtp *SMTPService,
) RequestEmailUpdateSource {
	return &struct {
		*CreateShortCodeService
		*SMTPService
	}{
		CreateShortCodeService: createShortCode,
		SMTPService:            smtp,
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
	source RequestEmailUpdateSource
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func NewRequestEmailUpdateService(source RequestEmailUpdateSource) *RequestEmailUpdateService {
	return &RequestEmailUpdateService{source: source}
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
	span := sentry.StartSpan(ctx, "RequestEmailUpdateService.RequestEmailUpdate")
	defer span.Finish()

	span.SetData("request.id", request.ID.String())
	span.SetData("request.email", request.Email)
	span.SetData("request.lang", request.Lang)

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(span.Context(), CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageValidateMail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL,
		Override: true,
	})
	if err != nil {
		span.SetData("service.createShortCode.error", err.Error())

		return nil, NewErrRequestEmailUpdateService(fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(span.Context()), request, shortCode)

	return shortCode, nil
}

func (service *RequestEmailUpdateService) sendMail(
	ctx context.Context, request RequestEmailUpdateRequest, shortCode *models.ShortCode,
) {
	span := sentry.StartSpan(ctx, "RequestEmailUpdateService.sendMail")
	defer span.Finish()

	defer service.wg.Done()

	service.source.SMTP(span.Context(), mails.Mails.EmailUpdate, request.Lang, []string{request.Email}, map[string]any{
		"ShortCode": shortCode.PlainCode,
		"Target":    request.ID.String(),
		"URL":       config.SMTP.URLs.UpdateEmail,
		"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL.String(),
	})
}
