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

//go:embed pg.mailOutboxReschedule.sql
var mailOutboxRescheduleQuery string

// ErrMailOutboxRescheduleNotFound is returned by [PgMailOutboxReschedule.Exec] when the entry is gone.
var ErrMailOutboxRescheduleNotFound = errors.New("mail outbox entry not found")

// MailOutboxRescheduleRequest is the input to [PgMailOutboxReschedule.Exec].
type MailOutboxRescheduleRequest struct {
	// ShortCodeID identifies the entry to reschedule.
	ShortCodeID uuid.UUID
	// DueAt is when the next attempt may start.
	DueAt time.Time
}

// PgMailOutboxReschedule sets when an entry's next attempt may start.
type PgMailOutboxReschedule struct{}

func NewPgMailOutboxReschedule() *PgMailOutboxReschedule {
	return &PgMailOutboxReschedule{}
}

func (dao *PgMailOutboxReschedule) Exec(
	ctx context.Context, request *MailOutboxRescheduleRequest,
) (*MailOutboxEntry, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgMailOutboxReschedule")
	defer span.End()

	span.SetAttributes(attribute.String("short_code.id", request.ShortCodeID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &MailOutboxEntry{}

	err = db.NewRaw(mailOutboxRescheduleQuery, request.ShortCodeID, request.DueAt).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrMailOutboxRescheduleNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
