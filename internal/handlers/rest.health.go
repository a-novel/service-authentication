package handlers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/textproto"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"
)

// smtpAuthenticationFailed is the SMTP reply code of a rejected login.
const smtpAuthenticationFailed = 535

// errSMTPUnhealthy is the error of a failed SMTP probe. The probe's span exports it, so it never
// wraps the server's reply text, which may contain credentials.
var errSMTPUnhealthy = errors.New("SMTP health check failed")

// RestHealthClientSMTP is the SMTP sender whose reachability the health check probes.
type RestHealthClientSMTP = smtp.Sender

// RestHealthAPIJSONKeys is the JSON Keys client surface used to check its dependencies.
type RestHealthAPIJSONKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

// NewRestHealth returns the handler of the public health route, which reports PostgreSQL, the SMTP
// relay and the JSON Keys service as up or down, as [httpf.Health] answers.
func NewRestHealth(
	apiJSONKeys RestHealthAPIJSONKeys,
	clientSMTP RestHealthClientSMTP,
	logger logging.Log,
) http.HandlerFunc {
	return httpf.Health(map[string]func(ctx context.Context) error{
		"client:postgres": postgres.Health,
		"client:smtp":     newSMTPProbe(clientSMTP, logger),
		// JSON Keys fails Status with Unavailable when one of its own dependencies is down, so its
		// success is the whole verdict.
		"api:jsonKeys": func(ctx context.Context) error {
			_, err := apiJSONKeys.Status(ctx, &servicejsonkeys.StatusRequest{})

			return err
		},
	})
}

// newSMTPProbe returns the probe of the SMTP relay. A failure names its bounded diagnostics on the
// probe's span and in a warning through logger, never the server's reply text.
func newSMTPProbe(clientSMTP RestHealthClientSMTP, logger logging.Log) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		err := clientSMTP.Ping()
		if err == nil {
			return nil
		}

		category, code := smtpHealthFailure(err)
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("smtp.failure_category", category), attribute.Int("smtp.reply_code", code),
		)
		logger.Warn(ctx, fmt.Sprintf("SMTP health check failed: category=%s smtp_reply_code=%d", category, code))

		return errSMTPUnhealthy
	}
}

// smtpHealthFailure extracts only allowlisted diagnostics from wrapped errors.
// An unknown failure loses its detail, which could hold a secret.
func smtpHealthFailure(err error) (string, int) {
	network, ok := errors.AsType[net.Error](err)
	if ok && network.Timeout() {
		return "timeout", 0
	}

	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return "dns", 0
	}

	if _, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		return "tls", 0
	}

	if _, ok := errors.AsType[tls.RecordHeaderError](err); ok {
		return "tls", 0
	}

	if reply, ok := errors.AsType[*textproto.Error](err); ok {
		switch {
		case reply.Code == smtpAuthenticationFailed:
			return "authentication", reply.Code
		case reply.Code >= 400 && reply.Code <= 599:
			return "smtp_rejected", reply.Code
		}

		return "unknown", 0
	}

	if _, ok := errors.AsType[*net.OpError](err); ok {
		return "connection", 0
	}

	return "unknown", 0
}
