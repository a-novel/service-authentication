package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/uptrace/bun"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	"github.com/a-novel/golib/smtp"
	jkApiModels "github.com/a-novel/service-json-keys/models/api"

	"github.com/a-novel/service-authentication/models/api"
)

func (api *API) Ping(_ context.Context) (apimodels.PingRes, error) {
	return &apimodels.PingOK{Data: strings.NewReader("pong")}, nil
}

func (api *API) reportPostgres(ctx context.Context) apimodels.Dependency {
	ctx, span := otel.Tracer().Start(ctx, "api.reportPostgres")
	defer span.End()

	pg, err := postgres.GetContext(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		_ = otel.ReportError(span, fmt.Errorf("retrieve postgres context: invalid type %T", pg))

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	err = pgdb.Ping()
	if err != nil {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "postgres",
			Status: apimodels.DependencyStatusDown,
		}
	}

	otel.ReportSuccessNoContent(span)

	return apimodels.Dependency{
		Name:   "postgres",
		Status: apimodels.DependencyStatusUp,
	}
}

func (api *API) reportJSONKeys(ctx context.Context) apimodels.Dependency {
	ctx, span := otel.Tracer().Start(ctx, "api.reportJSONKeys")
	defer span.End()

	rawRes, err := api.JKClient.Ping(ctx)
	if err != nil {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "json-keys",
			Status: apimodels.DependencyStatusDown,
		}
	}

	_, ok := rawRes.(*jkApiModels.PingOK)
	if !ok {
		_ = otel.ReportError(span, fmt.Errorf("ping JSON keys: unexpected response type %T", rawRes))

		return apimodels.Dependency{
			Name:   "json-keys",
			Status: apimodels.DependencyStatusDown,
		}
	}

	otel.ReportSuccessNoContent(span)

	return apimodels.Dependency{
		Name:   "json-keys",
		Status: apimodels.DependencyStatusUp,
	}
}

func (api *API) reportSMTP(ctx context.Context) apimodels.Dependency {
	_, span := otel.Tracer().Start(ctx, "api.reportSMTP")
	defer span.End()

	err := api.SMTPClient.Ping()
	if err != nil && !errors.Is(err, smtp.ErrPingTestSender) {
		_ = otel.ReportError(span, err)

		return apimodels.Dependency{
			Name:   "smtp",
			Status: apimodels.DependencyStatusDown,
		}
	}

	otel.ReportSuccessNoContent(span)

	return apimodels.Dependency{
		Name:   "smtp",
		Status: apimodels.DependencyStatusUp,
	}
}

func (api *API) Healthcheck(ctx context.Context) (apimodels.HealthcheckRes, error) {
	return &apimodels.Health{
		Postgres: api.reportPostgres(ctx),
		JsonKeys: api.reportJSONKeys(ctx),
		SMTP:     api.reportSMTP(ctx),
	}, nil
}
