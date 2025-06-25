package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/config/mails"
	"github.com/a-novel/service-authentication/models"
	"github.com/getsentry/sentry-go"
	"log"
	"net/smtp"
	"os"
	"text/template"
)

var DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Lshortfile)

type SMTPService struct{}

func (service *SMTPService) SMTP(
	ctx context.Context, message *template.Template, lang models.Lang, tos []string, data any,
) {
	span := sentry.StartSpan(ctx, "SMTP")
	defer span.Finish()

	if lang == "" {
		lang = models.LangEN
	}

	span.SetData("lang", lang)
	span.SetData("tos", tos)
	span.SetData("template", message.Name())

	messageParsed, err := mails.ParseMailTemplate(message, lang, data)
	if err != nil {
		span.SetData("error", fmt.Sprintf("parse transactional email: %v", err))

		return
	}

	// Only in test mode.
	if config.SMTP.Sandbox {
		jsonData, err := json.Marshal(map[string]any{
			"message":             "send transactional email",
			"to":                  tos,
			"dynamicTemplateData": data,
		})
		if err != nil {
			span.SetData("error", fmt.Sprintf("marshal email data: %v", err))

			return
		}

		// Don't use Sentry logger because it appends a prefix to the message, which breaks parsing during tests.
		DebugLogger.Println(string(jsonData))

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
		span.SetData("error", fmt.Sprintf("send transactional email: %v", err))

		return
	}
}

func NewSMTPService() *SMTPService {
	return &SMTPService{}
}
