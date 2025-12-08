package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

type ShortCodeCreateRegisterService interface {
	Exec(ctx context.Context, request *ShortCodeCreateRequest) (*ShortCode, error)
}
type ShortCodeCreateRegisterSmtp = smtp.Sender

type ShortCodeCreateRegisterRequest struct {
	Email string `validate:"required,email"`
	Lang  string `validate:"required,langs"`
}

type ShortCodeCreateRegister struct {
	service          ShortCodeCreateRegisterService
	smtp             smtp.Sender
	shortCodesConfig config.ShortCodes
	smtpConfig       config.SmtpUrls

	wg sync.WaitGroup
}

func NewShortCodeCreateRegister(
	service ShortCodeCreateRegisterService,
	smtp smtp.Sender,
	shortCodesConfig config.ShortCodes,
	smtpConfig config.SmtpUrls,
) *ShortCodeCreateRegister {
	return &ShortCodeCreateRegister{
		service:          service,
		smtp:             smtp,
		shortCodesConfig: shortCodesConfig,
		smtpConfig:       smtpConfig,
	}
}

func (service *ShortCodeCreateRegister) Wait() {
	service.wg.Wait()
}

func (service *ShortCodeCreateRegister) Exec(
	ctx context.Context, request *ShortCodeCreateRegisterRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateRegister")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Create a new short code.
	shortCode, err := service.service.Exec(ctx, &ShortCodeCreateRequest{
		Usage:    ShortCodeUsageRegister,
		Target:   request.Email,
		TTL:      service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL,
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

func (service *ShortCodeCreateRegister) sendMail(
	ctx context.Context, request *ShortCodeCreateRegisterRequest, shortCode *ShortCode,
) {
	defer service.wg.Done()

	_, span := otel.Tracer().Start(ctx, "service.ShortCodeCreateRegister(sendMail)")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.email", request.Email),
		attribute.String("request.lang", request.Lang),
		attribute.String("short_code.target", shortCode.Target),
	)

	logger := otel.Logger()

	err := service.smtp.SendMail(
		smtp.MailUsers{{Email: request.Email}},
		mails.Mails.Register,
		request.Lang,
		map[string]any{
			"ShortCode": shortCode.PlainCode,
			"Target":    base64.RawURLEncoding.EncodeToString([]byte(request.Email)),
			"URL":       service.smtpConfig.Register,
			"Duration":  service.shortCodesConfig.Usages[ShortCodeUsageRegister].TTL.Hours(),
			"Banner":    assets.BannerBase64,
			"_Purpose":  "register",
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, otel.ReportError(span, err).Error())

		return
	}

	logger.InfoContext(ctx, "register request sent to "+request.Email)
	otel.ReportSuccessNoContent(span)
}
