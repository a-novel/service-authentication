package api

import (
	"context"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/uptrace/bun"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/lib"
)

func (api *API) Ping(_ context.Context) (codegen.PingRes, error) {
	return &codegen.PingOK{Data: strings.NewReader("pong")}, nil
}

func (api *API) reportPostgres(ctx context.Context) codegen.Dependency {
	logger := sentry.NewLogger(ctx)

	pg, err := lib.PostgresContext(ctx)
	if err != nil {
		logger.Errorf(ctx, "retrieve postgres context: %v", err)

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		logger.Errorf(ctx, "retrieve postgres context: invalid type %T", pg)

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	err = pgdb.Ping()
	if err != nil {
		logger.Errorf(ctx, "ping postgres: %v", err)

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
