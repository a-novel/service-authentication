package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/smtp"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
	"github.com/a-novel/service-authentication/models/mails/assets"
)

// RequestRegisterSource is the source used to perform the RequestRegisterService.RequestRegister action.
type RequestRegisterSource interface {
	CreateShortCode(ctx context.Context, request CreateShortCodeRequest) (*models.ShortCode, error)
	smtp.Sender
}

func NewRequestRegisterServiceSource(
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
	source           RequestRegisterSource
	shortCodesConfig config.ShortCodes
	smtpConfig       models.SMTPURLsConfig
	// Enable graceful shutdowns by waiting for all goroutines spanned by the service to finish.
	wg sync.WaitGroup
}

func NewRequestRegisterService(
	source RequestRegisterSource,
	shortCodesConfig config.ShortCodes,
	smtpConfig models.SMTPURLsConfig,
) *RequestRegisterService {
	return &RequestRegisterService{source: source, shortCodesConfig: shortCodesConfig, smtpConfig: smtpConfig}
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
	ctx, span := otel.Tracer().Start(ctx, "service.RequestRegister")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang.String()),
	)

	// Create a new short code.
	shortCode, err := service.source.CreateShortCode(ctx, CreateShortCodeRequest{
		Usage:    models.ShortCodeUsageRequestRegister,
		Target:   request.Email,
		TTL:      service.shortCodesConfig.Usages[models.ShortCodeUsageRequestRegister].TTL,
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

func (service *RequestRegisterService) sendMail(
	ctx context.Context, request RequestRegisterRequest, shortCode *models.ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.RequestRegister.sendMail")
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
			models.Mails.Register,
			request.Lang.String(),
			map[string]any{
				"ShortCode": shortCode.PlainCode,
				"Target":    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
				"URL":       service.smtpConfig.Register,
				"Duration":  service.shortCodesConfig.Usages[models.ShortCodeUsageRequestRegister].TTL.Hours(),
				"Banner":    assets.BannerBase64,
				"_Purpose":  "register",
			},
		)
	}()

	select {
	case err := <-c:
		if err != nil {
			logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

			return
		}

		logger.InfoContext(ctx, "register request sent to "+request.Email)
		otel.ReportSuccessNoContent(span)

		return
	case <-time.After(SMTPTimeout):
		err := otel.ReportError(span, fmt.Errorf("send email: %w", os.ErrDeadlineExceeded))
		logger.ErrorContext(ctx, err.Error())

		return
	}
}
