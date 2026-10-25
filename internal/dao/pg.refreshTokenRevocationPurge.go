package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.refreshTokenRevocationPurge.sql
var refreshTokenRevocationPurgeQuery string

// RefreshTokenRevocationPurgeRequest is the input to [PgRefreshTokenRevocationPurge.Exec].
type RefreshTokenRevocationPurgeRequest struct {
	// Limit bounds the rows one call deletes.
	Limit int
}

// RefreshTokenRevocationPurgeResult reports one [PgRefreshTokenRevocationPurge.Exec] call.
type RefreshTokenRevocationPurgeResult struct {
	// Purged counts the rows the call deleted. Fewer than the limit means no expired row is left.
	Purged int64
}

// PgRefreshTokenRevocationPurge deletes revocations past their [RefreshTokenRevocation.ExpiresAt],
// oldest first, at most Limit of them.
type PgRefreshTokenRevocationPurge struct{}

func NewPgRefreshTokenRevocationPurge() *PgRefreshTokenRevocationPurge {
	return &PgRefreshTokenRevocationPurge{}
}

func (dao *PgRefreshTokenRevocationPurge) Exec(
	ctx context.Context, request *RefreshTokenRevocationPurgeRequest,
) (*RefreshTokenRevocationPurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgRefreshTokenRevocationPurge")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(refreshTokenRevocationPurgeQuery, request.Limit).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	purged, err := res.RowsAffected()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return &RefreshTokenRevocationPurgeResult{Purged: purged}, nil
}
