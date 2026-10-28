package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.mailOutboxDelete.sql
var mailOutboxDeleteQuery string

// ErrMailOutboxDeleteNotFound is returned by [PgMailOutboxDelete.Exec] when the entry is already gone.
var ErrMailOutboxDeleteNotFound = errors.New("mail outbox entry not found")

// MailOutboxDeleteRequest is the input to [PgMailOutboxDelete.Exec].
type MailOutboxDeleteRequest struct {
	// ShortCodeID identifies the entry to delete.
	ShortCodeID uuid.UUID
}

// PgMailOutboxDelete ends an entry. It holds the recipient's address, so nothing keeps it once no
// attempt is left.
type PgMailOutboxDelete struct{}

func NewPgMailOutboxDelete() *PgMailOutboxDelete {
	return &PgMailOutboxDelete{}
}

func (dao *PgMailOutboxDelete) Exec(ctx context.Context, request *MailOutboxDeleteRequest) (*MailOutboxEntry, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgMailOutboxDelete")
	defer span.End()

	span.SetAttributes(attribute.String("short_code.id", request.ShortCodeID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &MailOutboxEntry{}

	err = db.NewRaw(mailOutboxDeleteQuery, request.ShortCodeID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrMailOutboxDeleteNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
