package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/smtp"
)

const (
	RestHealthStatusUp   = "up"
	RestHealthStatusDown = "down"
)

// RestHealthStatus is the JSON representation of a single dependency's health.
// /healthcheck is unauthenticated and public, so the response carries no error
// message: raw errors routinely include internal hostnames, ports, or schema names.
// Operators read the underlying error off the trace span.
type RestHealthStatus struct {
	// Status is either RestHealthStatusUp or RestHealthStatusDown.
	Status string `json:"status"`
}

// NewRestHealthStatus converts an error into a RestHealthStatus, mapping nil to
// RestHealthStatusUp and any non-nil error to RestHealthStatusDown. The error itself
// stays out of the public response; see [RestHealthStatus].
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

// RestHealthClientSmtp is the SMTP sender whose reachability the health check probes.
type RestHealthClientSmtp = smtp.Sender

// RestHealthApiJsonKeys is the JSON-keys client surface the health check needs: a
// Status probe confirming the upstream service answers.
type RestHealthApiJsonKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

// RestHealth is the handler backing /healthcheck. It probes each downstream
// dependency and reports their combined status; see RestHealthStatus for why the
// response withholds error detail.
type RestHealth struct {
	apiJsonKeys RestHealthApiJsonKeys
	clientSmtp  RestHealthClientSmtp
}

// NewRestHealth returns a RestHealth handler wired to the dependencies it probes.
func NewRestHealth(
	apiJsonKeys RestHealthApiJsonKeys,
	clientSmtp RestHealthClientSmtp,
) *RestHealth {
	return &RestHealth{
		apiJsonKeys: apiJsonKeys,
		clientSmtp:  clientSmtp,
	}
}

func (handler *RestHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Health")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
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
		// Inside a transaction the handle exposes no pool to ping, so there is
		// nothing to probe; report healthy.
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

	_, err := handler.apiJsonKeys.Status(ctx, new(servicejsonkeys.StatusRequest))
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportSmtp(ctx context.Context) error {
	_, span := otel.Tracer().Start(ctx, "rest.Health(reportSmtp)")
	defer span.End()

	err := handler.clientSmtp.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
