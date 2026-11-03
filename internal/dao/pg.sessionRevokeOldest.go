package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionRevokeOldest.sql
var sessionRevokeOldestQuery string

// SessionRevokeOldestRequest is the input to [PgSessionRevokeOldest.Exec].
type SessionRevokeOldestRequest struct {
	// UserID is the account whose sessions to revoke.
	UserID uuid.UUID
	// Keep is how many active sessions the account keeps, the ones active last.
	Keep int
	// Limit bounds the sessions one call revokes.
	Limit int

	// Leeway extends each revocation past the end of its session.
	Leeway time.Duration
	// Now is the timestamp recorded as the revocation time.
	Now time.Time
}

// PgSessionRevokeOldest revokes the active sessions of an account past the Keep active last, the least
// recent first, at most Limit of them, as [PgSessionRevokeOthers] revokes. They then list as logged
// out.
type PgSessionRevokeOldest struct{}

func NewPgSessionRevokeOldest() *PgSessionRevokeOldest {
	return &PgSessionRevokeOldest{}
}

func (dao *PgSessionRevokeOldest) Exec(ctx context.Context, request *SessionRevokeOldestRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionRevokeOldest")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(
		sessionRevokeOldestQuery,
		request.UserID, request.Keep, request.Limit, request.Leeway.Seconds(), request.Now,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
