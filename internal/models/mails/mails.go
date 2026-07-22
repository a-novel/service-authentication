// Package mails holds the transactional email templates. Each email type is embedded
// once per supported language and parsed at initialization into a ready-to-render
// template set that the sending code selects by language.
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

// The templates read their values from a data map keyed by these names.
const (
	TemplateVarShortCode = "ShortCode"
	TemplateVarTarget    = "Target"
	TemplateVarURL       = "URL"
	TemplateVarDuration  = "Duration"
	TemplateVarBanner    = "Banner"
	TemplateVarPurpose   = "_Purpose"
)

// MailTemplates groups the parsed email templates by type. Each field carries every
// language variant as named sub-templates, chosen by language when the mail is rendered.
type MailTemplates struct {
	EmailUpdate   *template.Template
	PasswordReset *template.Template
	Register      *template.Template
}

// Mails holds the ready-to-render email templates, parsed once at package
// initialization. A malformed template panics at startup.
var Mails = MailTemplates{
	EmailUpdate:   template.Must(template.New(config.LangEN).Parse(emailUpdateEn)),
	PasswordReset: template.Must(template.New(config.LangEN).Parse(passwordResetEn)),
	Register:      template.Must(template.New(config.LangEN).Parse(registerEn)),
}

// Attach the French variant to each template as an associated sub-template, so one
// MailTemplates field can render either language. Only registering the variant matters,
// so the results are discarded.
var (
	_ = template.Must(Mails.EmailUpdate.New(config.LangFR).Parse(emailUpdateFr))
	_ = template.Must(Mails.PasswordReset.New(config.LangFR).Parse(passwordResetFr))
	_ = template.Must(Mails.Register.New(config.LangFR).Parse(registerFr))
)
