package dao_test

import (
	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/migrations"
)

var ctx context.Context

func init() {
	var err error

	// Share a global context, so the client is shared between tests and this prevents deadlocks.
	ctx, err = pgctx.NewContext(context.Background(), &migrations.Migrations)
	if err != nil {
		panic(err)
	}
}
