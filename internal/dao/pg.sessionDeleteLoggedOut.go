package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionDeleteLoggedOut.sql
var sessionDeleteLoggedOutQuery string

// SessionDeleteLoggedOutRequest is the input to [PgSessionDeleteLoggedOut.Exec].
type SessionDeleteLoggedOutRequest struct {
	// UserID is the account whose logged-out sessions to remove.
	UserID uuid.UUID
}

// PgSessionDeleteLoggedOut removes the row of every logged-out session of an account, in one
// statement. It never removes an active one, and an account without a logged-out session is no
// error.
type PgSessionDeleteLoggedOut struct{}

func NewPgSessionDeleteLoggedOut() *PgSessionDeleteLoggedOut {
	return &PgSessionDeleteLoggedOut{}
}

func (dao *PgSessionDeleteLoggedOut) Exec(ctx context.Context, request *SessionDeleteLoggedOutRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionDeleteLoggedOut")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(sessionDeleteLoggedOutQuery, request.UserID).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
