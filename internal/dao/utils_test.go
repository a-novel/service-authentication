package dao_test

import (
	"context"
	"os"

	"github.com/uptrace/bun/driver/pgdriver"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/service-authentication/migrations"
)

var ctx context.Context

func init() {
	var err error

	//nolint:fatcontext
	ctx, err = pgctx.NewContextWithOptions(
		context.Background(),
		&migrations.Migrations,
		pgdriver.WithDSN(os.Getenv("DAO_DSN")),
	)
	if err != nil {
		panic(err)
	}
}
