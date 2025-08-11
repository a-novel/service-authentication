package services

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
	"github.com/a-novel/service-authentication/models/mails/assets"
)

// RequestPasswordResetSource is the source used to perform the RequestPasswordResetService.RequestPasswordReset
// action.
type RequestPasswordResetSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	SelectCredentialsByEmail(ctx context.Context, email string) (*dao.CredentialsEntity, error)
	smtp.Sender
}

func NewRequestPasswordResetSource(
	selectCredentials *dao.SelectCredentialsByEmailRepository,
	createShortCode *CreateShortCodeService,
	smtpSender smtp.Sender,
) RequestPasswordResetSource {
	return &struct {
		*dao.SelectCredentialsByEmailRepository
		*CreateShortCodeService
		smtp.Sender
	}{
		SelectCredentialsByEmailRepository: selectCredentials,
		CreateShortCodeService:             createShortCode,
		Sender:                             smtpSender,
	}
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
	source           RequestPasswordResetSource
	shortCodesConfig config.ShortCodes
	smtpConfig       models.SMTPURLsConfig
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func NewRequestPasswordResetService(
	source RequestPasswordResetSource,
	shortCodesConfig config.ShortCodes,
	smtpConfig models.SMTPURLsConfig,
) *RequestPasswordResetService {
	return &RequestPasswordResetService{source: source, shortCodesConfig: shortCodesConfig, smtpConfig: smtpConfig}
}

func (service *RequestPasswordResetService) Wait() {
	service.wg.Wait()
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
	ctx, span := otel.Tracer().Start(ctx, "service.RequestPasswordReset")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang.String()),
	)

	credentials, err := service.source.SelectCredentialsByEmail(ctx, request.Email)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(ctx, CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageResetPassword,
		Target:   credentials.ID.String(),
		TTL:      service.shortCodesConfig.Usages[models.ShortCodeUsageResetPassword].TTL,
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

func (service *RequestPasswordResetService) sendMail(
	ctx context.Context, request RequestPasswordResetRequest, userID uuid.UUID, shortCode *models.ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.RequestPasswordReset.sendMail")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang.String()),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	c := make(chan error, 1)

	go func() {
		c <- service.source.SendMail(
			smtp.MailUsers{{Email: request.Email}},
			models.Mails.PasswordReset,
			request.Lang.String(),
			map[string]any{
				"ShortCode": shortCode.PlainCode,
				"Target":    userID.String(),
				"URL":       service.smtpConfig.UpdatePassword,
				"Duration":  service.shortCodesConfig.Usages[models.ShortCodeUsageResetPassword].TTL.Hours(),
				"Banner":    assets.BannerBase64,
				"_Purpose":  "password-reset",
			},
		)
	}()

	select {
	case err := <-c:
		if err != nil {
			logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

			return
		}

		logger.InfoContext(ctx, "password reset request sent to "+request.Email)
		otel.ReportSuccessNoContent(span)

		return
	case <-time.After(SMTPTimeout):
		err := otel.ReportError(span, fmt.Errorf("send email: %w", os.ErrDeadlineExceeded))
		logger.ErrorContext(ctx, err.Error())

		return
	}
}
