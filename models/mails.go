package models

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed mails/fr/email-update.mjml
	emailUpdateFr string
	//go:embed mails/fr/password-reset.mjml
	passwordResetFr string
	//go:embed mails/fr/register.mjml
	registerFr string
)

var (
	//go:embed mails/en/email-update.mjml
	emailUpdateEn string
	//go:embed mails/en/password-reset.mjml
	passwordResetEn string
	//go:embed mails/en/register.mjml
	registerEn string
)

type MailTemplates struct {
	EmailUpdate   *template.Template
	PasswordReset *template.Template
	Register      *template.Template
}

var Mails = MailTemplates{
	EmailUpdate:   template.Must(template.New(LangEN.String()).Parse(emailUpdateEn)),
	PasswordReset: template.Must(template.New(LangEN.String()).Parse(passwordResetEn)),
	Register:      template.Must(template.New(LangEN.String()).Parse(registerEn)),
}

var (
	_ = template.Must(Mails.EmailUpdate.New(LangFR.String()).Parse(emailUpdateFr))
	_ = template.Must(Mails.PasswordReset.New(LangFR.String()).Parse(passwordResetFr))
	_ = template.Must(Mails.Register.New(LangFR.String()).Parse(registerFr))
)
