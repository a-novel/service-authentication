package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionUpsert.sql
var sessionUpsertQuery string

// SessionUpsertRequest is the input to [PgSessionUpsert.Exec].
type SessionUpsertRequest struct {
	// See [Session.ID].
	ID uuid.UUID
	// See [Session.UserID].
	UserID uuid.UUID

	// See [Session.UserAgent].
	UserAgent string
	// IPAddress is the device's IP address. Empty records none.
	IPAddress string
	// Region is the device's CLDR region code. Empty records none.
	Region string
	// City is the device's city. Empty records none.
	City string

	// Now is the session's last activity, and its start when the call opens it.
	Now time.Time
	// See [Session.ExpiresAt]. It applies only when the call opens the session.
	ExpiresAt time.Time
}

// PgSessionUpsert records an access token issued for a session, with the device it went to. It
// opens the session's row when none exists, and otherwise replaces the device and the last
// activity.
type PgSessionUpsert struct{}

func NewPgSessionUpsert() *PgSessionUpsert {
	return &PgSessionUpsert{}
}

func (dao *PgSessionUpsert) Exec(ctx context.Context, request *SessionUpsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionUpsert")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(
		sessionUpsertQuery,
		request.ID,
		request.UserID,
		request.UserAgent,
		bun.NullZero(request.IPAddress),
		bun.NullZero(request.Region),
		bun.NullZero(request.City),
		request.Now,
		request.ExpiresAt,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
