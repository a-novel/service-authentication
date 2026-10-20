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
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"

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
		closePostgres bool
		transaction   bool
		cancel        bool
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
			name:       "Error/Smtp/Authentication",
			smtpError:  &textproto.Error{Code: 535, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=authentication smtp_reply_code=535",
		},
		{
			name:       "Error/Smtp/Rejected",
			smtpError:  &textproto.Error{Code: 454, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=smtp_rejected smtp_reply_code=454",
		},
		{
			name:       "Error/Smtp/InvalidReplyCode",
			smtpError:  &textproto.Error{Code: 123456789, Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=unknown smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/Timeout",
			smtpError:  &net.DNSError{Err: privateDetail, IsTimeout: true},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=timeout smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/DNS",
			smtpError:  &net.DNSError{Err: privateDetail, Name: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=dns smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/Certificate",
			smtpError:  &tls.CertificateVerificationError{Err: dependencyErr},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=tls smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/TLSRecord",
			smtpError:  tls.RecordHeaderError{Msg: privateDetail},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=tls smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/Connection",
			smtpError:  &net.OpError{Op: "dial", Err: dependencyErr},
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=connection smtp_reply_code=0",
		},
		{
			name:       "Error/Smtp/Unknown",
			smtpError:  dependencyErr,
			statusCode: http.StatusServiceUnavailable,
			states:     [3]string{"up", "down", "up"},
			smtpLog:    "SMTP health check failed: category=unknown smtp_reply_code=0",
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
			smtpLog:       "SMTP health check failed: category=unknown smtp_reply_code=0",
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
					defer func() {
						require.NoError(t, ctx.Err())
						require.NoError(t, tx.Rollback())
					}()

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

			smtpError := testCase.smtpError
			if smtpError != nil {
				// Wrapped detail must stay out of the diagnostics too.
				smtpError = fmt.Errorf("%s: %w", privateDetail, smtpError)
			}

			smtpClient.EXPECT().Ping().Return(smtpError).Once()
			// The payload is never inspected: an empty response reads as up.
			jsonKeysClient.EXPECT().Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Run(func(ctx context.Context, _ *servicejsonkeys.StatusRequest, _ ...grpc.CallOption) {
					if testCase.cancel {
						require.ErrorIs(t, ctx.Err(), context.Canceled)
					} else {
						require.NoError(t, ctx.Err())
					}
				}).
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
			smtpClient.AssertExpectations(t)
			jsonKeysClient.AssertExpectations(t)
		})
	}
}

//nolint:paralleltest // Replaces the process-wide tracer provider while probes run.
func TestRestHealthTelemetry(t *testing.T) {
	testCases := []struct {
		name         string
		smtpErr      error
		expectStatus int
	}{
		{name: "Success", expectStatus: http.StatusOK},
		{
			name:         "Error/Smtp",
			smtpErr:      &textproto.Error{Code: 535, Msg: "private SMTP reply"},
			expectStatus: http.StatusServiceUnavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
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

			smtpClient.EXPECT().Ping().Return(testCase.smtpErr).Once()
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
				Return(&servicejsonkeys.StatusResponse{}, nil).Once()

			handler := handlers.NewRestHealth(jsonKeysClient, smtpClient, &loggingpresets.LogLocal{Out: io.Discard})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil))

			require.Equal(t, testCase.expectStatus, w.Code)

			ended := recorder.Ended()
			require.Len(t, ended, 4)

			for _, span := range ended {
				expected := codes.Unset
				if testCase.smtpErr != nil && (span.Name() == "rest.Health" || span.Name() == "rest.Health(reportSmtp)") {
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
