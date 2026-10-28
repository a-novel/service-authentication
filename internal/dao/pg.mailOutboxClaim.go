package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.mailOutboxClaim.sql
var mailOutboxClaimQuery string

// ErrMailOutboxClaimNotFound is returned by [PgMailOutboxClaim.Exec] when the entry is gone or not
// due, as when another attempt holds it.
var ErrMailOutboxClaimNotFound = errors.New("mail outbox entry not found")

// MailOutboxClaimRequest is the input to [PgMailOutboxClaim.Exec].
type MailOutboxClaimRequest struct {
	// ShortCodeID identifies the entry to claim.
	ShortCodeID uuid.UUID
	// Now is the instant the entry must be due by.
	Now time.Time
	// LeaseUntil is the entry's new due time. No other attempt claims it before then.
	LeaseUntil time.Time
}

// PgMailOutboxClaim takes a due entry for one attempt. It counts the attempt and moves the entry's
// due time to the end of a lease, in one statement, so two attempts never claim it together.
type PgMailOutboxClaim struct{}

func NewPgMailOutboxClaim() *PgMailOutboxClaim {
	return &PgMailOutboxClaim{}
}

func (dao *PgMailOutboxClaim) Exec(ctx context.Context, request *MailOutboxClaimRequest) (*MailOutboxEntry, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgMailOutboxClaim")
	defer span.End()

	span.SetAttributes(attribute.String("short_code.id", request.ShortCodeID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &MailOutboxEntry{}

	err = db.NewRaw(mailOutboxClaimQuery, request.ShortCodeID, request.Now, request.LeaseUntil).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrMailOutboxClaimNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
