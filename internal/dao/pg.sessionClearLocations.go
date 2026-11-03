package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.sessionClearLocations.sql
var sessionClearLocationsQuery string

// SessionClearLocationsRequest is the input to [PgSessionClearLocations.Exec].
type SessionClearLocationsRequest struct {
	// UserID is the account whose sessions to clear.
	UserID uuid.UUID
}

// PgSessionClearLocations erases the IP address, the region and the city of every session of an
// account, in one statement.
type PgSessionClearLocations struct{}

func NewPgSessionClearLocations() *PgSessionClearLocations {
	return &PgSessionClearLocations{}
}

func (dao *PgSessionClearLocations) Exec(ctx context.Context, request *SessionClearLocationsRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgSessionClearLocations")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	_, err = db.NewRaw(sessionClearLocationsQuery, request.UserID).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}
