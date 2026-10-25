package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.refreshTokenRevocationInsert.sql
var refreshTokenRevocationInsertQuery string

// RefreshTokenRevocationInsertRequest is the input to [PgRefreshTokenRevocationInsert.Exec].
type RefreshTokenRevocationInsertRequest struct {
	// See [RefreshTokenRevocation.JTI].
	JTI string
	// See [RefreshTokenRevocation.ExpiresAt].
	ExpiresAt time.Time
	// Now is the timestamp recorded as the revocation time.
	Now time.Time
}

// PgRefreshTokenRevocationInsert revokes a refresh token. Revoking a token already revoked
// succeeds and keeps the first row.
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

	_, err = db.NewRaw(refreshTokenRevocationInsertQuery, request.JTI, request.ExpiresAt, request.Now).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
