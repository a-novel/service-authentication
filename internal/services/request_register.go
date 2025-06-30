package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/a-novel/service-authentication/models"
)

var ErrRequestRegisterService = errors.New("RequestRegisterService.RequestRegister")

func NewErrRequestRegisterService(err error) error {
	return errors.Join(err, ErrRequestRegisterService)
}

// RequestRegisterSource is the source used to perform the RequestRegisterService.RequestRegister action.
type RequestRegisterSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	SMTP(ctx context.Context, message *template.Template, lang models.Lang, tos []string, data any)
}

func NewRequestRegisterServiceSource(
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

// RequestRegisterRequest is the input used to perform the RequestRegisterService.RequestRegister action.
type RequestRegisterRequest struct {
	// Email of the user trying to register. This email will receive a link that can be used to register.
	Email string
	// Language of the account.
	Lang models.Lang
}

// RequestRegisterService is the service used to perform the RequestRegisterService.RequestRegister action.
//
// You may create one using the NewRequestRegisterService function.
type RequestRegisterService struct {
	source RequestRegisterSource
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func NewRequestRegisterService(source RequestRegisterSource) *RequestRegisterService {
	return &RequestRegisterService{source: source}
}

func (service *RequestRegisterService) Wait() {
	service.wg.Wait()
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
	span := sentry.StartSpan(ctx, "RequestRegisterService.RequestRegister")
	defer span.Finish()

	span.SetData("request.email", request.Email)
	span.SetData("request.lang", request.Lang)

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(span.Context(), CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageRequestRegister,
		Target:   request.Email,
		TTL:      config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL,
		Override: true,
	})
	if err != nil {
		span.SetData("dao.createShortCode.error", err.Error())

		return nil, NewErrRequestRegisterService(fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)

	go service.sendMail(context.WithoutCancel(span.Context()), request, shortCode)

	return shortCode, nil
}

func (service *RequestRegisterService) sendMail(
	ctx context.Context, request RequestRegisterRequest, shortCode *models.ShortCode,
) {
	span := sentry.StartSpan(ctx, "RequestRegisterService.sendMail")
	defer span.Finish()

	defer service.wg.Done()

	service.source.SMTP(span.Context(), mails.Mails.Register, request.Lang, []string{request.Email}, map[string]any{
		"ShortCode": shortCode.PlainCode,
		"Target":    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
		"URL":       config.SMTP.URLs.Register,
		"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageRequestRegister].TTL.String(),
	})
}
