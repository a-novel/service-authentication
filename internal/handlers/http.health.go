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
// The shape is deliberately minimal: /healthcheck is an unauthenticated public
// endpoint, so it must not expose raw error messages — those routinely include
// internal hostnames, ports, or schema names that leak infrastructure topology.
// The underlying error is recorded on the trace span for operators instead.
type RestHealthStatus struct {
	// Status is either RestHealthStatusUp or RestHealthStatusDown.
	Status string `json:"status"`
}

// NewRestHealthStatus converts an error into a RestHealthStatus, mapping nil to
// RestHealthStatusUp and any non-nil error to RestHealthStatusDown. The error
// itself is intentionally discarded from the public response; see RestHealthStatus.
func NewRestHealthStatus(err error) *RestHealthStatus {
	return &RestHealthStatus{
		Status: lo.Ternary(err == nil, RestHealthStatusUp, RestHealthStatusDown),
	}
}

type RestHealthClientSmtp = smtp.Sender

type RestHealthApiJsonKeys interface {
	Status(
		ctx context.Context,
		req *servicejsonkeys.StatusRequest,
		opts ...grpc.CallOption,
	) (*servicejsonkeys.StatusResponse, error)
}

type RestHealth struct {
	apiJsonKeys RestHealthApiJsonKeys
	clientSmtp  RestHealthClientSmtp
}

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
	ctx, span := otel.Tracer().Start(r.Context(), "rest.RestHealth")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
		"client:postgres": NewRestHealthStatus(handler.reportPostgres(ctx)),
		"client:smtp":     NewRestHealthStatus(handler.reportSmtp(ctx)),
		"api:jsonKeys":    NewRestHealthStatus(handler.reportJsonKeys(ctx)),
	})
}

func (handler *RestHealth) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "rest.RestHealth(reportPostgres)")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		// Cannot assess db connection if we are running on transaction mode
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
	ctx, span := otel.Tracer().Start(ctx, "rest.RestHealth(reportJsonKeys)")
	defer span.End()

	_, err := handler.apiJsonKeys.Status(ctx, new(servicejsonkeys.StatusRequest))
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *RestHealth) reportSmtp(ctx context.Context) error {
	_, span := otel.Tracer().Start(ctx, "rest.RestHealth(reportSmtp)")
	defer span.End()

	err := handler.clientSmtp.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
