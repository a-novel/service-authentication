package mails_test

import (
	"bytes"
	"html"
	"io"
	"mime"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/models/mails"
	"github.com/a-novel/service-authentication/v2/internal/models/mails/assets"
)

func TestMailTemplates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		template *template.Template
		language string
		subject  string
		source   string
	}{
		{
			name:     "Register/English",
			template: mails.Mails.Register,
			language: "en",
			subject:  "Create your Agora Storyverse account",
		},
		{
			name:     "Register/French",
			template: mails.Mails.Register,
			language: "fr",
			subject:  "Créez votre compte Agora Storyverse",
		},
		{
			name:     "PasswordReset/English",
			template: mails.Mails.PasswordReset,
			language: "en",
			subject:  "Reset your Agora Storyverse password",
		},
		{
			name:     "PasswordReset/French",
			template: mails.Mails.PasswordReset,
			language: "fr",
			subject:  "Réinitialisez votre mot de passe Agora Storyverse",
		},
		{
			name:     "EmailUpdate/English",
			template: mails.Mails.EmailUpdate,
			language: "en",
			subject:  "Confirm your new Agora Storyverse email address",
			source:   "bmV3QGV4YW1wbGUuY29t",
		},
		{
			name:     "EmailUpdate/French",
			template: mails.Mails.EmailUpdate,
			language: "fr",
			subject:  "Confirmez votre nouvelle adresse e-mail Agora Storyverse",
			source:   "bmV3QGV4YW1wbGUuY29t",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			const (
				baseURL   = "https://account.example.com/continue"
				shortCode = "preview-only-7N2Q4R"
				target    = "cmVhZGVyQGV4YW1wbGUuY29t"
			)

			data := map[string]any{
				mails.TemplateVarURL:       baseURL,
				mails.TemplateVarShortCode: shortCode,
				mails.TemplateVarTarget:    target,
				mails.TemplateVarDuration:  0.5,
				mails.TemplateVarBanner:    assets.BannerBase64,
				"Source":                   testCase.source,
			}

			var rendered bytes.Buffer

			err := testCase.template.ExecuteTemplate(&rendered, testCase.language, data)
			require.NoError(t, err)

			message, err := mail.ReadMessage(&rendered)
			require.NoError(t, err)
			require.Equal(t, testCase.subject, message.Header.Get("Subject"))
			require.Equal(t, "1.0", message.Header.Get("MIME-Version"))
			mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.Equal(t, "text/html", mediaType)
			require.Equal(t, "UTF-8", params["charset"])

			body, err := io.ReadAll(message.Body)
			require.NoError(t, err)

			content := string(body)
			require.Contains(t, content, `lang="`+testCase.language+`"`)
			require.Contains(t, content, "0.5 h.")
			require.Contains(t, content, "Agora Storyverse")
			require.Contains(t, content, `alt="Agora Storyverse"`)
			require.Contains(t, content, `src="data:image/png;base64,`)
			require.Equal(t, 1, strings.Count(content, "<h1 "))
			require.NotContains(t, content, "<no value>")
			require.NotContains(t, content, "{{")

			links := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(content, -1)
			require.Len(t, links, 2, "the primary action and copyable fallback must both be present")
			require.Equal(t, links[0][1], links[1][1])
			link, err := url.Parse(html.UnescapeString(links[0][1]))
			require.NoError(t, err)

			expectedQuery := url.Values{"shortCode": {shortCode}, "target": {target}}
			if testCase.source != "" {
				expectedQuery.Set("source", testCase.source)
			}

			require.Equal(t, expectedQuery, link.Query())
			link.RawQuery = ""
			require.Equal(t, baseURL, link.String())
		})
	}
}
