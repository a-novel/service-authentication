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
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var ErrRequestPasswordResetService = errors.New("RequestPasswordResetService.RequestPasswordReset")

func NewErrRequestPasswordResetService(err error) error {
	return errors.Join(err, ErrRequestPasswordResetService)
}

// RequestPasswordResetSource is the source used to perform the RequestPasswordResetService.RequestPasswordReset
// action.
type RequestPasswordResetSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	SelectCredentialsByEmail(ctx context.Context, email string) (*dao.CredentialsEntity, error)
	SMTP(ctx context.Context, message *template.Template, lang models.Lang, tos []string, data any)
}

// RequestPasswordResetRequest is the input used to perform the RequestPasswordResetService.RequestPasswordReset
// action.
type RequestPasswordResetRequest struct {
	// Email of the user trying to reset its password.
	Email string
	// Language of the account.
	Lang models.Lang
}

// RequestPasswordResetService is the service used to perform the RequestPasswordResetService.RequestPasswordReset
// action.
//
// You may create one using the NewRequestPasswordResetService function.
type RequestPasswordResetService struct {
	source RequestPasswordResetSource
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func (service *RequestPasswordResetService) Wait() {
	service.wg.Wait()
}

func (service *RequestPasswordResetService) sendMail(
	ctx context.Context, request RequestPasswordResetRequest, userID uuid.UUID, shortCode *models.ShortCode,
) {
	span := sentry.StartSpan(ctx, "RequestPasswordResetService.sendMail")
	defer span.Finish()

	defer service.wg.Done()

	service.source.SMTP(span.Context(), mails.Mails.EmailUpdate, request.Lang, []string{request.Email}, map[string]any{
		"ShortCode": shortCode.PlainCode,
		"Target":    userID.String(),
		"URL":       config.SMTP.URLs.UpdatePassword,
		"Duration":  config.ShortCodes.Usages[models.ShortCodeUsageResetPassword].TTL.String(),
	})
}

// RequestPasswordReset sends a short code to the user's email, allowing them to register.
//
// This indirect registration method ensures the email is valid on account creation. There is no need to put a new
// account in stale mode until its email is verified with this method.
//
// Once requested, the user MUST send the register form using the short code it received, otherwise account creation
// will fail.
func (service *RequestPasswordResetService) RequestPasswordReset(
	ctx context.Context, request RequestPasswordResetRequest,
) (*models.ShortCode, error) {
	span := sentry.StartSpan(ctx, "RequestPasswordResetService.RequestPasswordReset")
	defer span.Finish()

	span.SetData("request.email", request.Email)
	span.SetData("request.lang", request.Lang)

	credentials, err := service.source.SelectCredentialsByEmail(span.Context(), request.Email)
	if err != nil {
		span.SetData("dao.selectCredentialsByEmail.error", err.Error())

		return nil, NewErrRequestPasswordResetService(fmt.Errorf("check email existence: %w", err))
	}

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(span.Context(), CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      config.ShortCodes.Usages[models.ShortCodeUsageResetPassword].TTL,
		Override: true,
	})
	if err != nil {
		span.SetData("service.createShortCode.error", err.Error())

		return nil, NewErrRequestPasswordResetService(fmt.Errorf("create short code: %w", err))
	}

	// Sends the short code by mail, once the request is done (context terminated).
	service.wg.Add(1)
	go service.sendMail(context.WithoutCancel(span.Context()), request, credentials.ID, shortCode)

	return shortCode, nil
}

func NewRequestPasswordResetSource(
	selectCredentials *dao.SelectCredentialsByEmailRepository,
	createShortCode *CreateShortCodeService,
	smtp *SMTPService,
) RequestPasswordResetSource {
	return &struct {
		*dao.SelectCredentialsByEmailRepository
		*CreateShortCodeService
		*SMTPService
	}{
		SelectCredentialsByEmailRepository: selectCredentials,
		CreateShortCodeService:             createShortCode,
		SMTPService:                        smtp,
	}
}

func NewRequestPasswordResetService(source RequestPasswordResetSource) *RequestPasswordResetService {
	return &RequestPasswordResetService{source: source}
}
