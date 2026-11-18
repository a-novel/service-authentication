package config_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppPresetDefault(t *testing.T) {
	t.Parallel()

	// The token paths name projected files, not credentials.
	//
	//nolint:gosec // G101 reads the "Token" field names as secrets.
	testCases := []struct {
		name string

		restTimeoutRead          string
		restRateLimitStrict      string
		serviceJSONKeysTokenPath string

		expect    string
		expectErr string
	}{
		{name: "Success/Default", expect: "15s 3/5m0s {Host:localhost Port:8080 TLS:false TokenPath:}"},
		{
			name:                "Success/Override",
			restTimeoutRead:     "2s",
			restRateLimitStrict: "2/1m",
			expect:              "2s 2/1m0s {Host:localhost Port:8080 TLS:false TokenPath:}",
		},
		{
			name:                     "Success/ServiceToken",
			serviceJSONKeysTokenPath: "/tokens/json-keys",
			expect:                   "15s 3/5m0s {Host:localhost Port:8080 TLS:false TokenPath:/tokens/json-keys}",
		},
		{
			name:            "Error/InvalidRestTimeoutRead",
			restTimeoutRead: "invalid",
			expectErr:       `time: invalid duration "invalid"`,
		},
		{
			name:                "Error/InvalidRateLimit",
			restRateLimitStrict: "3",
			expectErr:           "invalid rate",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			command := exec.CommandContext(t.Context(), "go", "run", "./testdata/app")

			command.Env = append(os.Environ(),
				"GOWORK=off",
				"SERVICE_AUTHENTICATION_ENV_PREFIX=AUTH_CONFIG_TEST_",
				"REST_TIMEOUT_READ=invalid",
				"AUTH_CONFIG_TEST_REST_TIMEOUT_READ="+testCase.restTimeoutRead,
				"AUTH_CONFIG_TEST_REST_RATE_LIMIT_STRICT="+testCase.restRateLimitStrict,
				"AUTH_CONFIG_TEST_SERVICE_JSON_KEYS_TOKEN_PATH="+testCase.serviceJSONKeysTokenPath,
			)

			output, err := command.CombinedOutput()
			if testCase.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, string(output), testCase.expectErr)

				return
			}

			require.NoError(t, err, string(output))
			require.Equal(t, testCase.expect, strings.TrimSpace(string(output)))
		})
	}
}
