package lib

import (
	"context"
	"fmt"
	"net/smtp"
	"text/template"

	"github.com/rs/zerolog"

	sentryctx "github.com/a-novel-kit/context/sentry"

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
		sentryctx.CaptureException(ctx, fmt.Errorf("parse transactional email: %w", err))

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
		sentryctx.CaptureException(ctx, fmt.Errorf("send transactional email: %w", err))

		return
	}
}
