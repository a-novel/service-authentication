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
		serviceTokenPath         string
		serviceJSONKeysTLS       string

		expect    string
		expectErr string
	}{
		{name: "Success/Default", expect: "15s 3/5m0s *grpcf.LocalCredentialsProvider &{}"},
		{
			name:                "Success/Override",
			restTimeoutRead:     "2s",
			restRateLimitStrict: "2/1m",
			expect:              "2s 2/1m0s *grpcf.LocalCredentialsProvider &{}",
		},
		{
			name:                     "Success/ServiceToken",
			serviceJSONKeysTokenPath: "/var/run/secrets/tokens/json-keys",
			expect: "15s 3/5m0s *grpcf.ServiceTokenCredentialsProvider " +
				"&{ServiceTokenPath:/var/run/secrets/tokens/json-keys TLS:false}",
		},
		{
			name:                     "Success/ServiceTokenTLS",
			serviceJSONKeysTokenPath: "/var/run/secrets/tokens/json-keys",
			serviceJSONKeysTLS:       "true",
			expect: "15s 3/5m0s *grpcf.ServiceTokenCredentialsProvider " +
				"&{ServiceTokenPath:/var/run/secrets/tokens/json-keys TLS:true}",
		},
		{
			// SERVICE_TOKEN_PATH, the deprecated name, applies while the new one is unset.
			name:             "Success/DeprecatedServiceTokenPath",
			serviceTokenPath: "/var/run/secrets/tokens/legacy",
			expect: "15s 3/5m0s *grpcf.ServiceTokenCredentialsProvider " +
				"&{ServiceTokenPath:/var/run/secrets/tokens/legacy TLS:false}",
		},
		{
			name:                     "Success/ServiceTokenPathPrecedence",
			serviceJSONKeysTokenPath: "/var/run/secrets/tokens/json-keys",
			serviceTokenPath:         "/var/run/secrets/tokens/legacy",
			expect: "15s 3/5m0s *grpcf.ServiceTokenCredentialsProvider " +
				"&{ServiceTokenPath:/var/run/secrets/tokens/json-keys TLS:false}",
		},
		{
			// A local stack has no service token, so it keeps a plaintext connection.
			name:               "Success/TLSWithoutServiceToken",
			serviceJSONKeysTLS: "true",
			expect:             "15s 3/5m0s *grpcf.LocalCredentialsProvider &{}",
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
				"AUTH_CONFIG_TEST_SERVICE_TOKEN_PATH="+testCase.serviceTokenPath,
				"AUTH_CONFIG_TEST_SERVICE_JSON_KEYS_TLS="+testCase.serviceJSONKeysTLS,
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
