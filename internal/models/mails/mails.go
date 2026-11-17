// Package mails holds the transactional email templates. Each email type is embedded
// once per supported language and parsed at initialization into a ready-to-render
// template set that the sending code selects by language.
package mails

import (
	"embed"
	"encoding/base64"
	"mime"
	"slices"
	"strings"
	"text/template"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// templates holds every email, one HTML file per language directory.
//
//go:embed */*.html
var templates embed.FS

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

// Link is the data the templates carrying a short-code link read.
type Link struct {
	// ShortCode is the plaintext short code the link carries.
	ShortCode string
	// Target identifies the subject the short code is bound to, in the link.
	Target string
	// Source is the new address of an email change, base64url-encoded in the link.
	Source string
	// URL is the platform page the link opens.
	URL string
	// Duration is the short code's lifetime, in hours.
	Duration float64
}

// funcs are the functions the templates call. headerText writes a header value as RFC 2047
// encoded-words when it holds non-ASCII text: a header field body is ASCII (RFC 5322 §2.2) unless the
// relay announces SMTPUTF8 (RFC 6532). ASCII text passes unchanged.
var funcs = template.FuncMap{
	"headerText": func(text string) string { return mime.QEncoding.Encode("utf-8", text) },
}

// MailTemplates groups the parsed email templates by type. Each field carries every
// language variant as named sub-templates, chosen by language when the mail is rendered.
type MailTemplates struct {
	// EmailChanged tells the former address of an account that its email changed. It reads no value.
	EmailChanged  *template.Template
	EmailUpdate   *template.Template
	PasswordReset *template.Template
	Register      *template.Template
}

// Mails holds the ready-to-render email templates, parsed once at package
// initialization. A malformed template panics at startup.
var Mails = MailTemplates{
	EmailChanged:  parse("email-changed"),
	EmailUpdate:   parse("email-update"),
	PasswordReset: parse("password-reset"),
	Register:      parse("register"),
}

// parse reads every language variant of one email into a template set, each variant a
// sub-template named after its language, so a new language touches only the known languages and
// the template sources.
func parse(name string) *template.Template {
	set := template.New(name).Funcs(funcs)

	for _, lang := range authconfig.KnownLangs {
		source, err := templates.ReadFile(lang + "/" + name + ".html")
		if err != nil {
			panic(err)
		}

		template.Must(set.New(lang).Parse(string(source) + imageParts))
	}

	return set
}

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
