package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionPurge.sql
var sessionPurgeQuery string

// SessionPurgeRequest is the input to [PgSessionPurge.Exec].
type SessionPurgeRequest struct {
	// EndedBefore selects the sessions whose [Session.ExpiresAt] precedes it.
	EndedBefore time.Time
	// Limit bounds the rows one call deletes.
	Limit int
}

// SessionPurgeResult reports one [PgSessionPurge.Exec] call.
type SessionPurgeResult struct {
	// Purged counts the rows the call deleted. Fewer than the limit means no such session is left.
	Purged int64
}

// PgSessionPurge deletes the sessions that ended before a time, the earliest first, at most Limit of
// them.
type PgSessionPurge struct{}

func NewPgSessionPurge() *PgSessionPurge {
	return &PgSessionPurge{}
}

func (dao *PgSessionPurge) Exec(ctx context.Context, request *SessionPurgeRequest) (*SessionPurgeResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionPurge")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(sessionPurgeQuery, request.EndedBefore, request.Limit).Exec(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	purged, err := res.RowsAffected()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return &SessionPurgeResult{Purged: purged}, nil
}
