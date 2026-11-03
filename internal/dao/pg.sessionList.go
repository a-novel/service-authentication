package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionList.sql
var sessionListQuery string

// SessionListRequest is the input to [PgSessionList.Exec].
type SessionListRequest struct {
	// UserID is the account whose sessions to list.
	UserID uuid.UUID
	// Limit bounds the sessions the call returns.
	Limit int
}

// PgSessionList returns the sessions of an account, active and logged out: the active ones first, then
// latest activity first.
type PgSessionList struct{}

func NewPgSessionList() *PgSessionList {
	return &PgSessionList{}
}

func (dao *PgSessionList) Exec(ctx context.Context, request *SessionListRequest) ([]*Session, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := make([]*Session, 0, request.Limit)

	err = db.NewRaw(sessionListQuery, request.UserID, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
