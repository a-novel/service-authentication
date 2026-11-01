package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.refreshTokenRevocationInsert.sql
var refreshTokenRevocationInsertQuery string

// RefreshTokenRevocationInsertRequest is the input to [PgRefreshTokenRevocationInsert.Exec].
type RefreshTokenRevocationInsertRequest struct {
	// JTIs are the [RefreshTokenRevocation.JTI] values to revoke, all with the same expiry.
	JTIs []string
	// See [RefreshTokenRevocation.ExpiresAt].
	ExpiresAt time.Time
	// Now is the timestamp recorded as the revocation time.
	Now time.Time
}

// PgRefreshTokenRevocationInsert revokes JTIs in one statement. Revoking a JTI already revoked
// succeeds and keeps its first row.
type PgRefreshTokenRevocationInsert struct{}

func NewPgRefreshTokenRevocationInsert() *PgRefreshTokenRevocationInsert {
	return &PgRefreshTokenRevocationInsert{}
}

func (dao *PgRefreshTokenRevocationInsert) Exec(
	ctx context.Context, request *RefreshTokenRevocationInsertRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgRefreshTokenRevocationInsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(
		refreshTokenRevocationInsertQuery, bun.List(request.JTIs), request.ExpiresAt, request.Now,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
