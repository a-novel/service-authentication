package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.authorizationCodePurge.sql
var authorizationCodePurgeQuery string

// AuthorizationCodePurgeRequest is the input to [PgAuthorizationCodePurge.Exec].
type AuthorizationCodePurgeRequest struct {
	// Limit bounds the rows one call deletes.
	Limit int
}

// AuthorizationCodePurgeResult reports one [PgAuthorizationCodePurge.Exec] call.
type AuthorizationCodePurgeResult struct {
	// Purged counts the rows the call deleted. Fewer than the limit means no expired row is left.
	Purged int64
}

// PgAuthorizationCodePurge deletes authorization codes past their [AuthorizationCode.ExpiresAt],
// oldest first, at most Limit of them.
type PgAuthorizationCodePurge struct{}

func NewPgAuthorizationCodePurge() *PgAuthorizationCodePurge {
	return &PgAuthorizationCodePurge{}
}

func (dao *PgAuthorizationCodePurge) Exec(
	ctx context.Context, request *AuthorizationCodePurgeRequest,
) (*AuthorizationCodePurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgAuthorizationCodePurge")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(authorizationCodePurgeQuery, request.Limit).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	purged, err := res.RowsAffected()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return &AuthorizationCodePurgeResult{Purged: purged}, nil
}
