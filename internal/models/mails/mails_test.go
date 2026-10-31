package mails_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/models/mails"
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
			subject:  "Create your Agora account",
		},
		{
			name:     "Register/French",
			template: mails.Mails.Register,
			language: "fr",
			subject:  "Créez votre compte Agora",
		},
		{
			name:     "PasswordReset/English",
			template: mails.Mails.PasswordReset,
			language: "en",
			subject:  "Reset your Agora password",
		},
		{
			name:     "PasswordReset/French",
			template: mails.Mails.PasswordReset,
			language: "fr",
			subject:  "Réinitialisez votre mot de passe Agora",
		},
		{
			name:     "EmailUpdate/English",
			template: mails.Mails.EmailUpdate,
			language: "en",
			subject:  "Confirm your new Agora email address",
			source:   "bmV3QGV4YW1wbGUuY29t",
		},
		{
			name:     "EmailUpdate/French",
			template: mails.Mails.EmailUpdate,
			language: "fr",
			subject:  "Confirmez votre nouvelle adresse courriel Agora",
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
			require.Equal(t, "multipart/related", mediaType)
			require.Equal(t, "text/html", params["type"])

			parts := multipart.NewReader(message.Body, params["boundary"])

			htmlPart, err := parts.NextPart()
			require.NoError(t, err)
			mediaType, params, err = mime.ParseMediaType(htmlPart.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.Equal(t, "text/html", mediaType)
			require.Equal(t, "UTF-8", params["charset"])

			body, err := io.ReadAll(htmlPart)
			require.NoError(t, err)

			content := string(body)
			require.Contains(t, content, `lang="`+testCase.language+`"`)
			require.Contains(t, content, "0.5 h.")
			require.Contains(t, content, "Agora")
			require.Contains(t, content, `alt="Agora"`)
			require.Equal(t, 1, strings.Count(content, "<h1 "))

			// Clients block data URIs and list an unreferenced part as an attachment, so the HTML
			// shows exactly the images the message carries.
			images := regexp.MustCompile(`src="([^"]+)"`).FindAllStringSubmatch(content, -1)

			sources := make([]string, 0, len(images))
			for _, image := range images {
				sources = append(sources, image[1])
			}

			require.ElementsMatch(t, sources, readImages(t, parts))
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

// readImages reads the remaining parts of a message as PNG images and returns the cid: URL of each.
func readImages(t *testing.T, parts *multipart.Reader) []string {
	t.Helper()

	var images []string

	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			return images
		}

		require.NoError(t, err)
		require.Equal(t, "image/png", part.Header.Get("Content-Type"))
		require.Equal(t, "base64", part.Header.Get("Content-Transfer-Encoding"))

		image, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
		require.NoError(t, err)
		require.True(t, bytes.HasPrefix(image, []byte("\x89PNG\r\n\x1a\n")), "a part is not a PNG image")

		images = append(images, "cid:"+strings.Trim(part.Header.Get("Content-ID"), "<>"))
	}
}
