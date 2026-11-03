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

//go:embed pg.sessionTouch.sql
var sessionTouchQuery string

// SessionTouchRequest is the input to [PgSessionTouch.Exec].
type SessionTouchRequest struct {
	// See [Session.ID].
	ID uuid.UUID
	// See [Session.UserID].
	UserID uuid.UUID

	// Now is the session's last activity, and its start when the call opens it.
	Now time.Time
	// See [Session.ExpiresAt]. It applies only when the call opens the session.
	ExpiresAt time.Time
}

// PgSessionTouch records an access token issued for a session, and leaves its device as it was. It
// opens the session's row, with no device, when none exists.
type PgSessionTouch struct{}

func NewPgSessionTouch() *PgSessionTouch {
	return &PgSessionTouch{}
}

func (dao *PgSessionTouch) Exec(ctx context.Context, request *SessionTouchRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionTouch")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(sessionTouchQuery, request.ID, request.UserID, request.Now, request.ExpiresAt).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
