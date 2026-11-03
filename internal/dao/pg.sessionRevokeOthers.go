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

//go:embed pg.sessionRevokeOthers.sql
var sessionRevokeOthersQuery string

// SessionRevokeOthersRequest is the input to [PgSessionRevokeOthers.Exec].
type SessionRevokeOthersRequest struct {
	// UserID is the account whose sessions to revoke.
	UserID uuid.UUID
	// KeepID is the session the call leaves live.
	KeepID uuid.UUID

	// Leeway extends each revocation past the end of its session.
	Leeway time.Duration
	// Now is the timestamp recorded as the revocation time.
	Now time.Time
}

// PgSessionRevokeOthers revokes every session of an account that has not ended, but one, in one
// statement. A session already revoked keeps its first revocation.
type PgSessionRevokeOthers struct{}

func NewPgSessionRevokeOthers() *PgSessionRevokeOthers {
	return &PgSessionRevokeOthers{}
}

func (dao *PgSessionRevokeOthers) Exec(ctx context.Context, request *SessionRevokeOthersRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionRevokeOthers")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(
		sessionRevokeOthersQuery, request.UserID, request.KeepID, request.Leeway.Seconds(), request.Now,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
