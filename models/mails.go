package models

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed mails/fr/email-update.html
	emailUpdateFr string
	//go:embed mails/fr/password-reset.html
	passwordResetFr string
	//go:embed mails/fr/register.html
	registerFr string
)

var (
	//go:embed mails/en/email-update.html
	emailUpdateEn string
	//go:embed mails/en/password-reset.html
	passwordResetEn string
	//go:embed mails/en/register.html
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
