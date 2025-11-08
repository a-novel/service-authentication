package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	"github.com/uptrace/bun"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"
	jkApiModels "github.com/a-novel/service-json-keys/models/api"
)

const (
	HealthStatusUp   = "up"
	HealthStatusDown = "down"
)

type HeathStatus struct {
	Status string `json:"status"`
	Err    string `json:"err,omitempty"`
}

func NewHealthStatus(err error) *HeathStatus {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &HeathStatus{
		Status: lo.Ternary(err == nil, HealthStatusUp, HealthStatusDown),
		Err:    errMsg,
	}
}

type HealthClientSmtp = smtp.Sender

type HealthApiJsonkeys interface {
	Ping(ctx context.Context) (jkApiModels.PingRes, error)
}

type Health struct {
	apiJsonKeys HealthApiJsonkeys
	clientSmtp  HealthClientSmtp
}

func NewHealth(
	apiJsonKeys HealthApiJsonkeys,
	clientSmtp HealthClientSmtp,
) *Health {
	return &Health{
		apiJsonKeys: apiJsonKeys,
		clientSmtp:  clientSmtp,
	}
}

func (handler *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "api.Health")
	defer span.End()

	httpf.SendJSON(ctx, w, span, map[string]any{
		"client:postgres": NewHealthStatus(handler.reportPostgres(ctx)),
		"client:smtp":     NewHealthStatus(handler.reportSmtp(ctx)),
		"api:jsonKeys":    NewHealthStatus(handler.reportJsonKeys(ctx)),
	})
}

func (handler *Health) reportPostgres(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "api.Health(reportPostgres)")
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

func (handler *Health) reportJsonKeys(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "api.Health(reportJsonKeys)")
	defer span.End()

	rawRes, err := handler.apiJsonKeys.Ping(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	_, ok := rawRes.(*jkApiModels.PingOK)
	if !ok {
		return otel.ReportError(span, fmt.Errorf("ping JSON keys: unexpected response type %T", rawRes))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

func (handler *Health) reportSmtp(ctx context.Context) error {
	_, span := otel.Tracer().Start(ctx, "api.Health(reportSmtp)")
	defer span.End()

	err := handler.clientSmtp.Ping()
	if err != nil {
		return otel.ReportError(span, err)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
