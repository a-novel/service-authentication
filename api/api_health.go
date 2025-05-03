package api

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
	sentryctx "github.com/a-novel-kit/context/sentry"

	"github.com/a-novel/service-authentication/api/codegen"
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

func (api *API) Healthcheck(ctx context.Context) (codegen.HealthcheckRes, error) {
	return &codegen.Health{
		Postgres: api.reportPostgres(ctx),
	}, nil
}
