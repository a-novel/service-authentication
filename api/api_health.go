package api

import (
	"context"
	"fmt"
	sentrymiddleware "github.com/a-novel-kit/middlewares/sentry"
	"github.com/a-novel/service-authentication/internal/lib"
	"strings"

	"github.com/uptrace/bun"

	"github.com/a-novel/service-authentication/api/codegen"
)

func (api *API) Ping(_ context.Context) (codegen.PingRes, error) {
	return &codegen.PingOK{Data: strings.NewReader("pong")}, nil
}

func (api *API) reportPostgres(ctx context.Context) codegen.Dependency {
	pg, err := lib.PostgresContext(ctx)
	if err != nil {
		sentrymiddleware.CaptureError(ctx, fmt.Errorf("retrieve postgres context: %w", err))

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	pgdb, ok := pg.(*bun.DB)
	if !ok {
		sentrymiddleware.CaptureError(ctx, fmt.Errorf("retrieve postgres context: invalid type %T", pg))

		return codegen.Dependency{
			Name:   "postgres",
			Status: codegen.DependencyStatusDown,
		}
	}

	err = pgdb.Ping()
	if err != nil {
		sentrymiddleware.CaptureError(ctx, fmt.Errorf("ping postgres: %w", err))

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
