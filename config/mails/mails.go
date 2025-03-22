package mails

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/a-novel/authentication/models"
)

var (
	//go:embed fr/email-update.mjml
	emailUpdateFr string
	//go:embed fr/password-reset.mjml
	passwordResetFr string
	//go:embed fr/register.mjml
	registerFr string
)

var (
	//go:embed en/email-update.mjml
	emailUpdateEn string
	//go:embed en/password-reset.mjml
	passwordResetEn string
	//go:embed en/register.mjml
	registerEn string
)

type MailTemplates struct {
	EmailUpdate   *template.Template
	PasswordReset *template.Template
	Register      *template.Template
}

var Mails = MailTemplates{
	EmailUpdate:   template.Must(template.New(models.LangEN.String()).Parse(emailUpdateEn)),
	PasswordReset: template.Must(template.New(models.LangEN.String()).Parse(passwordResetEn)),
	Register:      template.Must(template.New(models.LangEN.String()).Parse(registerEn)),
}

var (
	_ = template.Must(Mails.EmailUpdate.New(models.LangFR.String()).Parse(emailUpdateFr))
	_ = template.Must(Mails.PasswordReset.New(models.LangFR.String()).Parse(passwordResetFr))
	_ = template.Must(Mails.Register.New(models.LangFR.String()).Parse(registerFr))
)

func ParseMailTemplate(t *template.Template, lang models.Lang, data any) (string, error) {
	builder := new(strings.Builder)

	if err := t.ExecuteTemplate(builder, lang.String(), data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return builder.String(), nil
}
