package handlers_test

import (
	"bytes"
	"context"
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
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/postgres"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

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

	dependencyErr := errors.New("private dependency detail")

	testCases := []struct {
		name          string
		skipPostgres  bool
		closePostgres bool
		transaction   bool
		cancel        bool
		smtpError     error
		jsonKeysError error
		statusCode    int
		states        [3]string
	}{
		{name: "Success", statusCode: http.StatusOK, states: [3]string{"up", "up", "up"}},
		{
			name:         "Error/MissingPostgres",
			skipPostgres: true,
			statusCode:   http.StatusServiceUnavailable,
			states:       [3]string{"down", "up", "up"},
		},
		{
			name:          "Error/ClosedPostgres",
			closePostgres: true,
			statusCode:    http.StatusServiceUnavailable,
			states:        [3]string{"down", "up", "up"},
		},
		{
			name:        "Error/TransactionContext",
			transaction: true,
			statusCode:  http.StatusServiceUnavailable,
			states:      [3]string{"down", "up", "up"},
		},
		{
			name:       "Error/CancelledProbe",
			cancel:     true,
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"down", "up", "up"},
		},
		{
			name:       "Error/Smtp",
			smtpError:  dependencyErr,
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
		},
		{
			name:          "Error/JsonKeys",
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
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			if !testCase.skipPostgres {
				var err error

				preset := postgrespresets.NewDefault(configtest.PostgresPreset.Options()...)
				ctx, err = postgres.NewContext(ctx, preset)
				require.NoError(t, err)
				pg, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				db, ok := pg.(*bun.DB)
				require.True(t, ok)
				t.Cleanup(func() { require.NoError(t, db.Close()) })

				if testCase.closePostgres {
					require.NoError(t, db.Close())
				}

				if testCase.transaction {
					tx, err := db.BeginTx(ctx, nil)
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, tx.Rollback()) })

					ctx = context.WithValue(ctx, postgres.ContextKey{}, tx)
				}
			}

			if testCase.cancel {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()

				ctx = cancelled
			}

			smtpClient := handlersmocks.NewMockRestHealthClientSmtp(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			smtpClient.EXPECT().Ping().Return(testCase.smtpError).Once()
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Run(func(ctx context.Context, _ *servicejsonkeys.StatusRequest, _ ...grpc.CallOption) {
					if testCase.cancel {
						require.ErrorIs(t, ctx.Err(), context.Canceled)
					} else {
						require.NoError(t, ctx.Err())
					}
				}).
				Return(jsonKeysHealth(t, 1), testCase.jsonKeysError).Once()

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil))
			require.Equal(t, testCase.statusCode, w.Code)
			require.JSONEq(t, fmt.Sprintf(`{
				"client:postgres":{"status":%q},
				"client:smtp":{"status":%q},
				"api:jsonKeys":{"status":%q}
			}`, testCase.states[0], testCase.states[1], testCase.states[2]), w.Body.String())
			require.NotContains(t, w.Body.String(), dependencyErr.Error())
			smtpClient.AssertExpectations(t)
			jsonKeysClient.AssertExpectations(t)
		})
	}
}

//nolint:paralleltest // Replaces the process-wide tracer provider while probes run.
func TestHealthTelemetry(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("Failed=%t", failed), func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := otel.GetTracerProvider()

			otel.SetTracerProvider(provider)
			t.Cleanup(func() {
				otel.SetTracerProvider(previous)
				require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
			})

			preset := postgrespresets.NewDefault(configtest.PostgresPreset.Options()...)
			ctx, err := postgres.NewContext(t.Context(), preset)
			require.NoError(t, err)
			pg, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			db, ok := pg.(*bun.DB)
			require.True(t, ok)
			t.Cleanup(func() { require.NoError(t, db.Close()) })

			smtpClient := handlersmocks.NewMockRestHealthClientSmtp(t)
			jsonKeysClient := handlersmocks.NewMockRestHealthApiJsonKeys(t)

			var smtpErr error
			if failed {
				smtpErr = &textproto.Error{Code: 535, Msg: "private SMTP reply"}
			}

			smtpClient.EXPECT().Ping().Return(smtpErr).Once()
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Run(func(ctx context.Context, _ *servicejsonkeys.StatusRequest, _ ...grpc.CallOption) {
					require.NoError(t, ctx.Err())

					var parent sdktrace.ReadWriteSpan

					for _, span := range recorder.Started() {
						if span.Name() == "rest.Health" {
							parent = span
						}
					}

					require.NotNil(t, parent)
					require.True(t, parent.IsRecording())
				}).
				Return(jsonKeysHealth(t, 1), nil).Once()

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil))

			if failed {
				require.Equal(t, http.StatusServiceUnavailable, w.Code)
			} else {
				require.Equal(t, http.StatusOK, w.Code)
			}

			ended := recorder.Ended()
			require.Len(t, ended, 4)

			for _, span := range ended {
				expected := codes.Ok
				if failed && (span.Name() == "rest.Health" || span.Name() == "rest.Health(reportSmtp)") {
					expected = codes.Error
				}

				require.Equal(t, expected, span.Status().Code, span.Name())
				require.NotContains(t, span.Status().Description, "private SMTP reply")

				for _, event := range span.Events() {
					for _, attr := range event.Attributes {
						require.NotContains(t, attr.Value.AsString(), "private SMTP reply")
					}
				}
			}

			smtpClient.AssertExpectations(t)
			jsonKeysClient.AssertExpectations(t)
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
			require.Equal(t, http.StatusServiceUnavailable, w.Code)
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
			require.Equal(t, http.StatusServiceUnavailable, w.Code)

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
