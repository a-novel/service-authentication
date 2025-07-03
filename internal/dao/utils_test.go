package dao_test

import (
	"context"
	"os"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/migrations"
)

var ctx context.Context

func init() {
	var err error

	//nolint:fatcontext
	ctx, err = lib.NewPostgresContext(context.Background(), os.Getenv("DAO_DSN"), migrations.Migrations)
	if err != nil {
		panic(err)
	}
}
