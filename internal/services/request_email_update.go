package services

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/models"
)

var ErrRequestEmailUpdateService = errors.New("RequestEmailUpdateService.RequestEmailUpdate")

func NewErrRequestEmailUpdateService(err error) error {
	return errors.Join(err, ErrRequestEmailUpdateService)
}

// RequestEmailUpdateSource is the source used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
type RequestEmailUpdateSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
}

// RequestEmailUpdateRequest is the input used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
type RequestEmailUpdateRequest struct {
	// ID of the account trying to update its email.
	ID uuid.UUID
	// New email of the account. This email will receive a link to confirm the email update.
	Email string
}

// RequestEmailUpdateService is the service used to perform the RequestEmailUpdateService.RequestEmailUpdate action.
//
// You may create one using the NewRequestEmailUpdateService function.
type RequestEmailUpdateService struct {
	source RequestEmailUpdateSource
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func (service *RequestEmailUpdateService) Wait() {
	service.wg.Wait()
}

func (service *RequestEmailUpdateService) sendMail(
	ctx context.Context, request RequestEmailUpdateRequest, shortCode *models.ShortCode,
) {
	defer service.wg.Done()

	// Send the mail.
	from := mail.NewEmail(config.Sendgrid.Sender.Name, config.Sendgrid.Sender.Mail)
	recipient := mail.NewEmail("", request.Email)

	message := mail.NewV3Mail()
	personalization := mail.NewPersonalization()

	personalization.AddTos(recipient)
	personalization.SetDynamicTemplateData(
		"duration", config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL.String(),
	)
	personalization.SetDynamicTemplateData("shortCode", shortCode.PlainCode)
	personalization.SetDynamicTemplateData("target", request.ID.String())

	message.SetFrom(from)
	message.SetTemplateID(config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].SendgridID)
	message.AddPersonalizations(personalization)

	lib.SendMail(ctx, message)
}

// RequestEmailUpdate requests an email update for a specific account.
//
// The link to update the email is sent to the new email address provided in the request. This also serves as
// a validation for the new email address.
func (service *RequestEmailUpdateService) RequestEmailUpdate(
	ctx context.Context, request RequestEmailUpdateRequest,
) (*models.ShortCode, error) {
	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(ctx, CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageValidateMail,
		Target:   request.ID.String(),
		Data:     request.Email,
		TTL:      config.ShortCodes.Usages[models.ShortCodeUsageValidateMail].TTL,
		Override: true,
	})
	if err != nil {
		return nil, NewErrRequestEmailUpdateService(fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)
	go service.sendMail(context.WithoutCancel(ctx), request, shortCode)

	return shortCode, nil
}

func NewRequestEmailUpdateService(source RequestEmailUpdateSource) *RequestEmailUpdateService {
	return &RequestEmailUpdateService{source: source}
}
