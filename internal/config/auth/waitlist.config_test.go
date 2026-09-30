package auth_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

func TestWaitlist(t *testing.T) {
	t.Parallel()

	validURL := "https://script.google.com/macros/s/test-deployment/exec"

	testCases := []struct {
		name    string
		url     string
		secret  string
		timeout time.Duration
		valid   bool
	}{
		{name: "Disabled", valid: true},
		{name: "Enabled", url: validURL, secret: strings.Repeat("a", 32), timeout: time.Second, valid: true},
		{name: "MissingSecret", url: validURL, timeout: time.Second},
		{name: "MissingURL", secret: strings.Repeat("a", 32), timeout: time.Second},
		{name: "WeakSecret", url: validURL, secret: "short", timeout: time.Second},
		{name: "Unbounded", url: validURL, secret: strings.Repeat("a", 32)},
		{name: "ExcessiveTimeout", url: validURL, secret: strings.Repeat("a", 32), timeout: time.Minute},
		{name: "MaximumTimeout", url: validURL, secret: strings.Repeat("a", 32), timeout: 20 * time.Second, valid: true},
		{name: "ShortSecretBoundary", url: validURL, secret: strings.Repeat("a", 31), timeout: time.Second},
		{name: "MalformedURL", url: ":%", secret: strings.Repeat("a", 32), timeout: time.Second},
		{name: "Fragment", url: validURL + "#fragment", secret: strings.Repeat("a", 32), timeout: time.Second},
		{
			name: "HTTP", url: "http://script.google.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
		},
		{
			name: "UntrustedHost", url: "https://example.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
		},
		{
			name: "EmbeddedCredentials", url: "https://user@script.google.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
		},
		{name: "Query", url: validURL + "?secret=private", secret: strings.Repeat("a", 32), timeout: time.Second},
		{
			name: "TestDeployment", url: "https://script.google.com/macros/s/test/dev",
			secret: strings.Repeat("a", 32), timeout: time.Second,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := authconfig.Waitlist{URL: testCase.url, Secret: testCase.secret, Timeout: testCase.timeout}
			if testCase.valid {
				require.NoError(t, cfg.Validate())
			} else {
				require.ErrorIs(t, cfg.Validate(), authconfig.ErrWaitlistConfig)
			}

			data, err := json.Marshal(cfg)
			require.NoError(t, err)
			require.NotContains(t, string(data), `"Secret"`)

			if testCase.secret != "" {
				require.NotContains(t, string(data), testCase.secret)
			}
		})
	}
}
