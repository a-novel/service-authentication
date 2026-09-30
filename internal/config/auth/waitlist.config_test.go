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
		name          string
		url           string
		secret        string
		timeout       time.Duration
		valid         bool
		expectMessage string
	}{
		{name: "Disabled", valid: true},
		{name: "Enabled", url: validURL, secret: strings.Repeat("a", 32), timeout: time.Second, valid: true},
		{
			name: "MissingSecret", url: validURL, timeout: time.Second,
			expectMessage: "URL and secret must both be set or both be empty",
		},
		{
			name: "MissingURL", secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL and secret must both be set or both be empty",
		},
		{
			name: "WeakSecret", url: validURL, secret: "short", timeout: time.Second,
			expectMessage: "secret must contain at least 32 bytes",
		},
		{
			name: "Unbounded", url: validURL, secret: strings.Repeat("a", 32),
			expectMessage: "timeout must be greater than zero and at most 20s",
		},
		{
			name: "ExcessiveTimeout", url: validURL, secret: strings.Repeat("a", 32), timeout: time.Minute,
			expectMessage: "timeout must be greater than zero and at most 20s",
		},
		{name: "MaximumTimeout", url: validURL, secret: strings.Repeat("a", 32), timeout: 20 * time.Second, valid: true},
		{
			name: "ShortSecretBoundary", url: validURL, secret: strings.Repeat("a", 31), timeout: time.Second,
			expectMessage: "secret must contain at least 32 bytes",
		},
		{
			name: "MalformedURL", url: ":%", secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL cannot be parsed",
		},
		{
			name: "MalformedURLWithCredentials", url: "https://user:private@script.google.com/%zz?secret=private",
			secret: strings.Repeat("a", 32), timeout: time.Second, expectMessage: "URL cannot be parsed",
		},
		{
			name: "Fragment", url: validURL + "#fragment", secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must not contain credentials, a query, or a fragment",
		},
		{
			name: "HTTP", url: "http://script.google.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must use https://script.google.com",
		},
		{
			name: "UntrustedHost", url: "https://example.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must use https://script.google.com",
		},
		{
			name: "EmbeddedCredentials", url: "https://user@script.google.com/macros/s/test/exec",
			secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must not contain credentials, a query, or a fragment",
		},
		{
			name: "Query", url: validURL + "?secret=private", secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must not contain credentials, a query, or a fragment",
		},
		{
			name: "TestDeployment", url: "https://script.google.com/macros/s/test/dev",
			secret: strings.Repeat("a", 32), timeout: time.Second,
			expectMessage: "URL must use /macros/s/<deployment>/exec",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := authconfig.Waitlist{URL: testCase.url, Secret: testCase.secret, Timeout: testCase.timeout}

			err := cfg.Validate()
			if testCase.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, authconfig.ErrWaitlistConfig)
				require.ErrorContains(t, err, testCase.expectMessage)
				require.NotContains(t, err.Error(), "private")

				if testCase.secret != "" {
					require.NotContains(t, err.Error(), testCase.secret)
				}
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
