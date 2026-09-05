package handlers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/textproto"

	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"
)

const (
	RestHealthStatusUp       = "up"
	RestHealthStatusDown     = "down"
	smtpAuthenticationFailed = 535
)

var (
	errJsonKeysUnhealthy = errors.New("JSON Keys PostgreSQL health is not UP")
	errSmtpUnhealthy     = errors.New("SMTP health check failed")
)

// RestHealthStatus is the JSON representation of a single dependency's health.
// /v2/healthcheck is unauthenticated and public, so the response carries no error
// message: raw errors routinely include internal hostnames, ports, or schema names.
// SMTP diagnostics use bounded categories in the configured logger and trace,
// never the server's reply text, which may contain credentials.
type RestHealthStatus struct {
	// Status is either RestHealthStatusUp or RestHealthStatusDown.
	Status string `json:"status"`
}

// NewRestHealthStatus maps nil to up and an error to down without exposing it.
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

// RestHealthClientSmtp is the SMTP sender whose reachability the health check probes.
type RestHealthClientSmtp = smtp.Sender

// RestHealthApiJsonKeys is the JSON Keys client surface used to check its dependencies.
type RestHealthApiJsonKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

// RestHealth backs /v2/healthcheck with public up/down states only.
type RestHealth struct {
	apiJsonKeys RestHealthApiJsonKeys
	clientSmtp  RestHealthClientSmtp
	logger      logging.Log
}

// NewRestHealth wires the probes and a logger that works without tracing enabled.
func NewRestHealth(
	apiJsonKeys RestHealthApiJsonKeys,
	clientSmtp RestHealthClientSmtp,
	logger logging.Log,
) *RestHealth {
	return &RestHealth{
		apiJsonKeys: apiJsonKeys,
		clientSmtp:  clientSmtp,
		logger:      logger,
	}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, map[string]any{
		"client:postgres": NewRestHealthStatus(handler.reportPostgres(ctx)),
		"client:smtp":     NewRestHealthStatus(handler.reportSmtp(ctx)),
		"api:jsonKeys":    NewRestHealthStatus(handler.reportJsonKeys(ctx)),
	})
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// A transaction exposes no pool to ping.
		return nil
	}

	err = pgdb.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportJsonKeys(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportJsonKeys)")
	defer span.End()

	response, err := handler.apiJsonKeys.Status(ctx, &servicejsonkeys.StatusRequest{})
	if err != nil {
		return otel.ReportError(span, err)
	}

	// The client exposes the response but not its enum constants. Match the
	// protocol's named UP value; missing and future unknown states fail closed.
	if response.GetPostgres().GetStatus().String() != "DEPENDENCY_STATUS_UP" {
		return otel.ReportError(span, errJsonKeysUnhealthy)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportSmtp(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.Health(reportSmtp)")
	defer span.End()

	err := handler.clientSmtp.Ping()
	if err != nil {
		category, code := smtpHealthFailure(err)
		span.SetAttributes(attribute.String("smtp.failure_category", category), attribute.Int("smtp.reply_code", code))
		handler.logger.Warn(ctx, fmt.Sprintf("SMTP health check failed: category=%s smtp_reply_code=%d", category, code))

		// Do not wrap the original error: tracing must not export SMTP reply text.
		return otel.ReportError(span, errSmtpUnhealthy)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

// smtpHealthFailure extracts only allowlisted diagnostics from wrapped errors.
// Unknown failures deliberately lose detail instead of risking a secret leak.
func smtpHealthFailure(err error) (string, int) {
	var (
		timeout     net.Error
		dns         *net.DNSError
		certificate *tls.CertificateVerificationError
		record      tls.RecordHeaderError
		reply       *textproto.Error
		network     *net.OpError
	)

	switch {
	case errors.As(err, &timeout) && timeout.Timeout():
		return "timeout", 0
	case errors.As(err, &dns):
		return "dns", 0
	case errors.As(err, &certificate), errors.As(err, &record):
		return "tls", 0
	case errors.As(err, &reply):
		if reply.Code == smtpAuthenticationFailed {
			return "authentication", reply.Code
		}

		if reply.Code >= 400 && reply.Code <= 599 {
			return "smtp_rejected", reply.Code
		}
	case errors.As(err, &network):
		return "connection", 0
	}

	return "unknown", 0
}
