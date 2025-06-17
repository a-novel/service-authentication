package lib

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/a-novel/service-authentication/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"time"
)

type PostgresKey struct{}

var ErrInvalidPostgresContext = errors.New("invalid postgres context")

const PostgresDSNEnv = "DSN"

const PingTimeout = 10 * time.Second

func NewPostgresContext(ctx context.Context, dsn string) (context.Context, error) {
	// Open a connection to the database.
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	// Make a temporary assignation. If something goes wrong, it is unnecessary and misleading to assign a value
	// to the global variable.
	client := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	// Wait for connection to be established.
	start := time.Now()
	for err := client.PingContext(ctx); err != nil; err = client.PingContext(ctx) {
		if time.Since(start) > PingTimeout {
			return nil, fmt.Errorf("ping database: %w", err)
		}
	}

	// Apply migrations.
	mig := migrate.NewMigrations()

	if err := mig.Discover(migrations.Migrations); err != nil {
		return nil, fmt.Errorf("discover mig: %w", err)
	}

	migrator := migrate.NewMigrator(client, mig)
	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	if _, err := migrator.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("apply mig: %w", err)
	}

	ctxPG := context.WithValue(ctx, PostgresKey{}, bun.IDB(client))
	// Close clients on context termination.
	context.AfterFunc(ctxPG, func() {
		_ = client.Close()
		_ = sqldb.Close()
	})

	return ctxPG, nil
}

func PostgresContext(ctx context.Context) (bun.IDB, error) {
	db, ok := ctx.Value(PostgresKey{}).(bun.IDB)
	if !ok {
		return nil, fmt.Errorf(
			"(pgctx) extract pg: %w: got type %T, expected %T",
			ErrInvalidPostgresContext,
			ctx.Value(PostgresKey{}), bun.IDB(nil),
		)
	}

	return db, nil
}

func PostgresContextTx(ctx context.Context, opts *sql.TxOptions) (context.Context, func(commit bool) error, error) {
	pg, err := PostgresContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("extract pg: %w", err)
	}

	tx, err := pg.BeginTx(ctx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}

	var done bool

	ctxTx, cancelFn := context.WithCancel(context.WithValue(ctx, PostgresKey{}, bun.IDB(&tx)))
	context.AfterFunc(ctxTx, func() {
		if !done {
			// If context is canceled without calling the cancel function, abort.
			// If the cancel function was already called, this will return an error,
			// so we ignore it.
			_ = tx.Rollback()
		}
	})

	cancelFnAugmented := func(commit bool) error {
		defer cancelFn()

		if commit {
			done = true

			return tx.Commit()
		}

		return nil
	}

	return ctxTx, cancelFnAugmented, nil
}
