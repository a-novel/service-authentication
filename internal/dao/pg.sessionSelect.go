package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionSelect.sql
var sessionSelectQuery string

// ErrSessionSelectNotFound is returned by [PgSessionSelect.Exec] when the account holds no session of
// the requested ID.
var ErrSessionSelectNotFound = errors.New("session not found")

// SessionSelectRequest is the input to [PgSessionSelect.Exec].
type SessionSelectRequest struct {
	ID uuid.UUID
	// UserID is the account the session must sign in.
	UserID uuid.UUID
}

// PgSessionSelect fetches a session of an account, active or logged out.
type PgSessionSelect struct{}

func NewPgSessionSelect() *PgSessionSelect {
	return &PgSessionSelect{}
}

func (dao *PgSessionSelect) Exec(ctx context.Context, request *SessionSelectRequest) (*Session, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &Session{}

	err = db.NewRaw(sessionSelectQuery, request.ID, request.UserID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrSessionSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
