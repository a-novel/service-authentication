package handlers_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func jsonKeysHealth(t *testing.T, status int) *servicejsonkeys.StatusResponse {
	t.Helper()

	response := &servicejsonkeys.StatusResponse{}
	require.NoError(t, protojson.Unmarshal(fmt.Appendf(nil, `{"postgres":{"status":%d}}`, status), response))

	return response
}

func TestHealth(t *testing.T) {
	t.Parallel()

	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("Failed=%t", failed), func(t *testing.T) {
			t.Parallel()

			smtpClient := handlersmocks.NewMockRestHealthClientSmtp(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			var dependencyErr error

			state := handlers.RestHealthStatusUp

			if failed {
				dependencyErr = errors.New("private dependency detail")
				state = handlers.RestHealthStatusDown
			}

			smtpClient.EXPECT().Ping().Return(dependencyErr).Once()
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(jsonKeysHealth(t, 1), dependencyErr).Once()
			ctx, err := postgres.NewContext(t.Context(), configtest.PostgresPreset)
			require.NoError(t, err)

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil))
			require.Equal(t, http.StatusOK, w.Code)
			require.JSONEq(t, fmt.Sprintf(`{
				"client:postgres":{"status":"up"},
				"client:smtp":{"status":%q},
				"api:jsonKeys":{"status":%q}
			}`, state, state), w.Body.String())
		})
	}
}

func TestHealthJsonKeysDependencies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response *servicejsonkeys.StatusResponse
		err      error
		state    string
	}{
		{name: "Up", response: jsonKeysHealth(t, 1), state: "up"},
		{name: "Down", response: jsonKeysHealth(t, 2), state: "down"},
		{name: "Unspecified", response: jsonKeysHealth(t, 0), state: "down"},
		{name: "Unknown", response: jsonKeysHealth(t, 99), state: "down"},
		{name: "MissingPostgres", response: &servicejsonkeys.StatusResponse{}, state: "down"},
		{name: "MissingResponse", state: "down"},
		{name: "RPCError", response: jsonKeysHealth(t, 1), err: errors.New("private RPC detail"), state: "down"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			smtpClient := handlersmocks.NewMockRestHealthClientSmtp(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			smtpClient.EXPECT().Ping().Return(nil).Once()
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(testCase.response, testCase.err).Once()
			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/healthcheck", nil))
			require.Equal(t, http.StatusOK, w.Code)
			require.JSONEq(t, fmt.Sprintf(`{
				"client:postgres":{"status":"down"},
				"client:smtp":{"status":"up"},
				"api:jsonKeys":{"status":%q}
			}`, testCase.state), w.Body.String())
		})
	}
}

func TestHealthSmtpDiagnostics(t *testing.T) {
	t.Parallel()

	const privateDetail = "fixture-secret-must-never-appear"

	testCases := []struct {
		name     string
		err      error
		category string
		code     int
	}{
		{name: "Healthy"},
		{name: "Authentication", err: &textproto.Error{Code: 535, Msg: privateDetail}, category: "authentication", code: 535},
		{name: "Rejected", err: &textproto.Error{Code: 454, Msg: privateDetail}, category: "smtp_rejected", code: 454},
		{name: "InvalidReplyCode", err: &textproto.Error{Code: 123456789, Msg: privateDetail}, category: "unknown"},
		{name: "Timeout", err: &net.DNSError{Err: privateDetail, IsTimeout: true}, category: "timeout"},
		{name: "DNS", err: &net.DNSError{Err: privateDetail, Name: privateDetail}, category: "dns"},
		{name: "Certificate", err: &tls.CertificateVerificationError{Err: errors.New(privateDetail)}, category: "tls"},
		{name: "TLSRecord", err: tls.RecordHeaderError{Msg: privateDetail}, category: "tls"},
		{name: "Connection", err: &net.OpError{Op: "dial", Err: errors.New(privateDetail)}, category: "connection"},
		{name: "Unknown", err: errors.New(privateDetail), category: "unknown"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			smtpClient := handlersmocks.NewMockRestHealthClientSmtp(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			var wrapped error
			if testCase.err != nil {
				wrapped = fmt.Errorf("%s: %w", privateDetail, testCase.err)
			}

			smtpClient.EXPECT().Ping().Return(wrapped).Once()
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(jsonKeysHealth(t, 1), nil).Once()

			var logs bytes.Buffer

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: &logs})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/healthcheck", nil))
			require.Equal(t, http.StatusOK, w.Code)

			var body map[string]handlers.RestHealthStatus
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

			if testCase.err == nil {
				require.Empty(t, logs.String())
				require.Equal(t, "up", body["client:smtp"].Status)
			} else {
				require.Equal(t, "down", body["client:smtp"].Status)
				require.Contains(t, logs.String(), "SMTP health check failed")
				require.Contains(t, logs.String(), testCase.category)
				require.Contains(t, logs.String(), fmt.Sprintf("smtp_reply_code=%d", testCase.code))
			}

			require.NotContains(t, logs.String(), privateDetail)
			require.NotContains(t, w.Body.String(), privateDetail)
		})
	}
}
