package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"github.com/sendgrid/sendgrid-go"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
	sentryctx "github.com/a-novel-kit/context/sentry"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/config"
)

func (api *API) Ping(_ context.Context) (codegen.PingRes, error) {
	return &codegen.PingOK{Data: strings.NewReader("pong")}, nil
}

func (api *API) reportPostgres(ctx context.Context) codegen.Dependency {
	logger := zerolog.Ctx(ctx)

	pg, err := pgctx.Context(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("retrieve context")
		sentryctx.CaptureException(ctx, err)

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		logger.Error().Msgf("invalid context type: %T", pg)
		sentryctx.CaptureMessage(ctx, fmt.Sprintf("invalid context type: %T", pg))

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	err = pgdb.Ping()
	if err != nil {
		logger.Error().Err(err).Msg("ping postgres")
		sentryctx.CaptureException(ctx, err)

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	return codegen.Dependency{
		Name:   "postgres",
		Status: codegen.DependencyStatusUp,
	}
}

func (api *API) reportSendgrid(ctx context.Context) codegen.Dependency {
	logger := zerolog.Ctx(ctx)

	request := sendgrid.GetRequest(config.Sendgrid.APIKey, "/v3/scopes", "https://api.sendgrid.com")
	request.Method = http.MethodGet

	response, err := sendgrid.API(request)
	if err != nil {
		logger.Error().Err(err).Msg("ping sendgrid")
		sentryctx.CaptureException(ctx, err)

		return codegen.Dependency{
			Name:   "sendgrid",
			Status: codegen.DependencyStatusDown,
		}
	}

	if response.StatusCode != http.StatusOK {
		logger.Error().
			Str("body", response.Body).
			Msgf("unexpected status code from sendgrid API: %d", response.StatusCode)

		sentryctx.CaptureMessage(ctx, fmt.Sprintf(
			"unexpected status code from sendgrid API: %d\n%s",
			response.StatusCode,
			response.Body,
		))

		return codegen.Dependency{
			Name:   "sendgrid",
			Status: codegen.DependencyStatusDown,
		}
	}

	return codegen.Dependency{
		Name:   "sendgrid",
		Status: codegen.DependencyStatusUp,
	}
}

func (api *API) Healthcheck(ctx context.Context) (codegen.HealthcheckRes, error) {
	return &codegen.Health{
		Postgres: api.reportPostgres(ctx),
		Sendgrid: api.reportSendgrid(ctx),
	}, nil
}
