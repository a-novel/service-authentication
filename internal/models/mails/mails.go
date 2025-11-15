package mails

import (
	_ "embed"
	"text/template"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

var (
	//go:embed fr/email-update.html
	emailUpdateFr string
	//go:embed fr/password-reset.html
	passwordResetFr string
	//go:embed fr/register.html
	registerFr string
)

var (
	//go:embed en/email-update.html
	emailUpdateEn string
	//go:embed en/password-reset.html
	passwordResetEn string
	//go:embed en/register.html
	registerEn string
)

type MailTemplates struct {
	EmailUpdate   *template.Template
	PasswordReset *template.Template
	Register      *template.Template
}

var Mails = MailTemplates{
	EmailUpdate:   template.Must(template.New(config.LangEN).Parse(emailUpdateEn)),
	PasswordReset: template.Must(template.New(config.LangEN).Parse(passwordResetEn)),
	Register:      template.Must(template.New(config.LangEN).Parse(registerEn)),
}

var (
	_ = template.Must(Mails.EmailUpdate.New(config.LangFR).Parse(emailUpdateFr))
	_ = template.Must(Mails.PasswordReset.New(config.LangFR).Parse(passwordResetFr))
	_ = template.Must(Mails.Register.New(config.LangFR).Parse(registerFr))
)
