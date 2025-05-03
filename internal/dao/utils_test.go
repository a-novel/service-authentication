package dao_test

import (
	"os"

	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/service-authentication/migrations"
)

var ctx context.Context

func init() {
	var err error

	ctx, err = pgctx.NewContextWithOptions(
		context.Background(),
		&migrations.Migrations,
		pgdriver.WithDSN(os.Getenv("DAO_DSN")),
	)
	if err != nil {
		panic(err)
	}
}
