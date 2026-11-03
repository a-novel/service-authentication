package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionTrim.sql
var sessionTrimQuery string

// SessionTrimRequest is the input to [PgSessionTrim.Exec].
type SessionTrimRequest struct {
	// UserID is the account whose sessions to trim.
	UserID uuid.UUID
	// Keep is how many logged-out sessions the account keeps, the ones active last.
	Keep int
	// Limit bounds the rows one call deletes.
	Limit int
}

// PgSessionTrim deletes the logged-out sessions of an account past the Keep active last, the least
// recent first, at most Limit of them. It never deletes an active session.
type PgSessionTrim struct{}

func NewPgSessionTrim() *PgSessionTrim {
	return &PgSessionTrim{}
}

func (dao *PgSessionTrim) Exec(ctx context.Context, request *SessionTrimRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionTrim")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(sessionTrimQuery, request.UserID, request.Keep, request.Limit).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
