package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/models"
)

var ErrRequestRegisterService = errors.New("RequestRegisterService.RequestRegister")

func NewErrRequestRegisterService(err error) error {
	return errors.Join(err, ErrRequestRegisterService)
}

// RequestRegisterSource is the source used to perform the RequestRegisterService.RequestRegister action.
type RequestRegisterSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
}

// RequestRegisterRequest is the input used to perform the RequestRegisterService.RequestRegister action.
type RequestRegisterRequest struct {
	// Email of the user trying to register. This email will receive a link that can be used to register.
	Email string
}

// RequestRegisterService is the service used to perform the RequestRegisterService.RequestRegister action.
//
// You may create one using the NewRequestRegisterService function.
type RequestRegisterService struct {
	source RequestRegisterSource
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func (service *RequestRegisterService) Wait() {
	service.wg.Wait()
}

func (service *RequestRegisterService) sendMail(
	parent context.Context, request RequestRegisterRequest, shortCode *models.ShortCode,
) {
	defer service.wg.Done()

	// Create a non-cancelable context from parent, so this method is still able to use the context after the parent
	// cancellation.
	ctx := context.WithoutCancel(parent)

	// Wait for parent context to be done.
	<-parent.Done()

	// Send the mail.
	from := mail.NewEmail(config.Sendgrid.Sender.Name, config.Sendgrid.Sender.Mail)
	recipient := mail.NewEmail("", request.Email)

	message := mail.NewV3Mail()
	personalization := mail.NewPersonalization()

	personalization.AddTos(recipient)
	personalization.SetDynamicTemplateData(
		"duration", config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL.String(),
	)
	personalization.SetDynamicTemplateData("shortCode", shortCode.PlainCode)
	personalization.SetDynamicTemplateData("target", base64.RawURLEncoding.EncodeToString([]byte(request.Email)))

	message.SetFrom(from)
	message.SetTemplateID(config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].SendgridID)
	message.AddPersonalizations(personalization)

	lib.SendMail(ctx, message)
}

// RequestRegister sends a short code to the user's email, allowing them to register.
//
// This indirect registration method ensures the email is valid on account creation. There is no need to put a new
// account in stale mode until its email is verified with this method.
//
// Once requested, the user MUST send the register form using the short code it received, otherwise account creation
// will fail.
func (service *RequestRegisterService) RequestRegister(
	ctx context.Context, request RequestRegisterRequest,
) (*models.ShortCode, error) {
	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(ctx, CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageRequestRegister,
		Target:   request.Email,
		TTL:      config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL,
		Override: true,
	})
	if err != nil {
		return nil, NewErrRequestRegisterService(fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)
	go service.sendMail(ctx, request, shortCode)

	return shortCode, nil
}

func NewRequestRegisterService(source RequestRegisterSource) *RequestRegisterService {
	return &RequestRegisterService{source: source}
}
