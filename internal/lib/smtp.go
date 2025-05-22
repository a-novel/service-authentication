package lib

import (
	"context"
	"fmt"
	sentrymiddleware "github.com/a-novel-kit/middlewares/sentry"
	"net/smtp"
	"text/template"

	"github.com/rs/zerolog"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/a-novel/service-authentication/models"
)

func SMTP(ctx context.Context, message *template.Template, lang models.Lang, tos []string, data any) {
	logger := zerolog.Ctx(ctx)

	if lang == "" {
		lang = models.LangEN
	}

	messageParsed, err := mails.ParseMailTemplate(message, lang, data)
	if err != nil {
		logger.Error().Err(err).Msg("parse transactional email")
		sentrymiddleware.CaptureError(ctx, fmt.Errorf("parse transactional email: %w", err))

		return
	}

	if config.SMTP.Sandbox {
		logger.Info().
			Strs("to", tos).
			Interface("dynamicTemplateData", data).
			Msg("send transactional email")

		return
	}

	err = smtp.SendMail(
		config.SMTP.Addr,
		smtp.PlainAuth(
			config.SMTP.Sender.Name,
			config.SMTP.Sender.Email,
			config.SMTP.Sender.Password,
			config.SMTP.Sender.Domain,
		),
		config.SMTP.Sender.Email,
		tos,
		[]byte(messageParsed),
	)
	if err != nil {
		logger.Error().Err(err).Msg("send transactional email")
		sentrymiddleware.CaptureError(ctx, fmt.Errorf("send transactional email: %w", err))

		return
	}
}
