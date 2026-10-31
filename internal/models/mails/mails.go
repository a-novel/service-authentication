// Package mails holds the transactional email templates. Each email type is embedded
// once per supported language and parsed at initialization into a ready-to-render
// template set that the sending code selects by language.
package mails

import (
	_ "embed"
	"encoding/base64"
	"slices"
	"strings"
	"text/template"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
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

// The banners are the 80h Agora exports of @a-novel-kit/ui-images, shown 40 pixels tall: the dark
// one by default, the light one when the mail client reports a light color scheme.
var (
	//go:embed assets/agora-banner.png
	banner []byte
	//go:embed assets/agora-banner-light.png
	bannerLight []byte
)

const (
	// boundary separates the parts of the multipart/related message each template opens.
	boundary = "agora-related"
	// base64LineLength is the longest encoded line RFC 2045 allows.
	base64LineLength = 76
)

// imageParts closes every message with the banners. Mail clients block data URIs, so the images
// travel as parts of the message and the HTML shows them by Content-ID.
var imageParts = imagePart("agora-banner@agorastoryverse.com", banner) +
	imagePart("agora-banner-light@agorastoryverse.com", bannerLight) +
	"\n--" + boundary + "--\n"

// The templates read their values from a data map keyed by these names.
const (
	// TemplateVarShortCode is the plaintext short code the link carries.
	TemplateVarShortCode = "ShortCode"
	// TemplateVarTarget identifies the subject the short code is bound to, in the link.
	TemplateVarTarget = "Target"
	// TemplateVarSource is the new address of an email change, base64url-encoded in the link.
	TemplateVarSource = "Source"
	// TemplateVarURL is the platform page the link opens.
	TemplateVarURL = "URL"
	// TemplateVarDuration is the short code's lifetime, in hours.
	TemplateVarDuration = "Duration"
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
	EmailUpdate:   template.Must(template.New(authconfig.LangEN).Parse(emailUpdateEn + imageParts)),
	PasswordReset: template.Must(template.New(authconfig.LangEN).Parse(passwordResetEn + imageParts)),
	Register:      template.Must(template.New(authconfig.LangEN).Parse(registerEn + imageParts)),
}

// Attach the French variant to each template as an associated sub-template, so one
// MailTemplates field can render either language. Only registering the variant matters,
// so the results are discarded.
var (
	_ = template.Must(Mails.EmailUpdate.New(authconfig.LangFR).Parse(emailUpdateFr + imageParts))
	_ = template.Must(Mails.PasswordReset.New(authconfig.LangFR).Parse(passwordResetFr + imageParts))
	_ = template.Must(Mails.Register.New(authconfig.LangFR).Parse(registerFr + imageParts))
)

// imagePart encodes a PNG as an inline MIME part named contentID.
func imagePart(contentID string, png []byte) string {
	var part strings.Builder

	part.WriteString("\n--" + boundary + "\nContent-Type: image/png\nContent-Transfer-Encoding: base64\n")
	part.WriteString("Content-Disposition: inline\nContent-ID: <" + contentID + ">\n")

	for line := range slices.Chunk([]byte(base64.StdEncoding.EncodeToString(png)), base64LineLength) {
		part.WriteByte('\n')
		part.Write(line)
	}

	return part.String()
}
