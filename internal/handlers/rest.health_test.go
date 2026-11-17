package handlers_test

import (
	"bytes"
	"context"
	"crypto/tls"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/postgres"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-authentication/v2/internal/handlers/mocks"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	const privateDetail = "private dependency detail"

	dependencyErr := errors.New(privateDetail)

	testCases := []struct {
		name          string
		skipPostgres  bool
		smtpError     error
		jsonKeysError error
		statusCode    int
		states        [3]string
		// smtpLog is the warning a failed SMTP probe logs; empty expects no log at all.
		smtpLog string
	}{
		{name: "Success", statusCode: http.StatusOK, states: [3]string{"up", "up", "up"}},
		{
			name:         "Error/MissingPostgres",
			skipPostgres: true,
			statusCode:   http.StatusServiceUnavailable,
			states:       [3]string{"down", "up", "up"},
		},
		{
			name:       "Error/SMTP/Authentication",
			smtpError:  &textproto.Error{Code: 535, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=authentication smtp_reply_code=535",
		},
		{
			name:       "Error/SMTP/Rejected",
			smtpError:  &textproto.Error{Code: 454, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=smtp_rejected smtp_reply_code=454",
		},
		{
			name:       "Error/SMTP/InvalidReplyCode",
			smtpError:  &textproto.Error{Code: 123456789, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=unknown smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/Timeout",
			smtpError:  &net.DNSError{Err: privateDetail, IsTimeout: true},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=timeout smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/DNS",
			smtpError:  &net.DNSError{Err: privateDetail, Name: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=dns smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/Certificate",
			smtpError:  &tls.CertificateVerificationError{Err: dependencyErr},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=tls smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/TLSRecord",
			smtpError:  tls.RecordHeaderError{Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=tls smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/Connection",
			smtpError:  &net.OpError{Op: "dial", Err: dependencyErr},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=connection smtp_reply_code=0",
		},
		{
			name:       "Error/SMTP/Unknown",
			smtpError:  dependencyErr,
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=unknown smtp_reply_code=0",
		},
		{
			name:          "Error/JSONKeys",
			jsonKeysError: dependencyErr,
			statusCode:    http.StatusServiceUnavailable,
			states:        [3]string{"up", "up", "down"},
		},
		{
			name:          "Error/All",
			skipPostgres:  true,
			smtpError:     dependencyErr,
			jsonKeysError: dependencyErr,
			statusCode:    http.StatusServiceUnavailable,
			states:        [3]string{"down", "down", "down"},
			smtpLog:       "SMTP health check failed: category=unknown smtp_reply_code=0",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			if !testCase.skipPostgres {
				ctx = newPostgresContext(t)
			}

			smtpClient := handlersmocks.NewMockRestHealthClientSMTP(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthAPIJSONKeys(t)

			smtpError := testCase.smtpError
			if smtpError != nil {
				// Wrapped detail must stay out of the diagnostics too.
				smtpError = fmt.Errorf("%s: %w", privateDetail, smtpError)
			}

			smtpClient.EXPECT().Ping().Return(smtpError).Once()
			// The payload is never inspected: an empty response reads as up.
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(&servicejsonkeys.StatusResponse{}, testCase.jsonKeysError).Once()

			var logs bytes.Buffer

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: &logs})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil))
			require.Equal(t, testCase.statusCode, w.Code)
			require.JSONEq(t, fmt.Sprintf(`{
				"client:postgres":{"status":%q},
				"client:smtp":{"status":%q},
				"api:jsonKeys":{"status":%q}
			}`, testCase.states[0], testCase.states[1], testCase.states[2]), w.Body.String())

			if testCase.smtpLog == "" {
				require.Empty(t, logs.String())
			} else {
				require.Contains(t, logs.String(), testCase.smtpLog)
			}

			require.NotContains(t, logs.String(), privateDetail)
			require.NotContains(t, w.Body.String(), privateDetail)
		})
	}
}

//nolint:paralleltest // Replaces the process-wide tracer provider while probes run.
func TestRestHealthTelemetry(t *testing.T) {
	const reply = "private SMTP reply"

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
	})

	smtpClient := handlersmocks.NewMockRestHealthClientSMTP(t)
	jsonKeysClient := handlersmocks.NewMockRestHealthAPIJSONKeys(t)

	smtpClient.EXPECT().Ping().Return(&textproto.Error{Code: 535, Msg: reply}).Once()
	jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
		Return(&servicejsonkeys.StatusResponse{}, nil).Once()

	handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/healthcheck", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var probe sdktrace.ReadOnlySpan

	for _, span := range recorder.Ended() {
		require.NotContains(t, span.Status().Description, reply, span.Name())

		for _, attr := range span.Attributes() {
			require.NotContains(t, attr.Value.String(), reply, span.Name())
		}

		if span.Name() == "rest.Health(client:smtp)" {
			probe = span
		}
	}

	// The probe's span describes the failure in bounded diagnostics.
	require.NotNil(t, probe)
	require.Equal(t, codes.Error, probe.Status().Code)
	require.Contains(t, probe.Attributes(), attribute.String("smtp.failure_category", "authentication"))
	require.Contains(t, probe.Attributes(), attribute.Int("smtp.reply_code", 535))
}

// newPostgresContext returns a context carrying a pool of its own on the test database, which closes
// when the test ends.
func newPostgresContext(t *testing.T) context.Context {
	t.Helper()

	preset := postgrespresets.NewDefault(configtest.PostgresPreset.Options()...)
	t.Cleanup(preset.Close)

	ctx, err := postgres.NewContext(t.Context(), preset)
	require.NoError(t, err)

	return ctx
}
