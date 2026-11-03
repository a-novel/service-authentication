package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionDelete.sql
var sessionDeleteQuery string

// ErrSessionDeleteNotFound is returned by [PgSessionDelete.Exec] when the account holds no logged-out
// session of the requested ID.
var ErrSessionDeleteNotFound = errors.New("logged-out session not found")

// SessionDeleteRequest is the input to [PgSessionDelete.Exec].
type SessionDeleteRequest struct {
	ID uuid.UUID
	// UserID is the account the session must sign in.
	UserID uuid.UUID
}

// PgSessionDelete removes the row of a logged-out session of an account. It never removes an active
// one.
type PgSessionDelete struct{}

func NewPgSessionDelete() *PgSessionDelete {
	return &PgSessionDelete{}
}

func (dao *PgSessionDelete) Exec(ctx context.Context, request *SessionDeleteRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionDelete")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(sessionDeleteQuery, request.ID, request.UserID).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	if deleted == 0 {
		return otel.ReportError(span, ErrSessionDeleteNotFound)
	}

	return nil
}
