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

//go:embed pg.shortCodeDelete.sql
var shortCodeDeleteQuery string

// ErrShortCodeDeleteNotFound is returned by [PgShortCodeDelete.Exec] when no active short code matches
// the requested ID.
var ErrShortCodeDeleteNotFound = errors.New("short code not found")

const (
	// ShortCodeDeleteOverride is the deletion comment set when a newer code supersedes
	// an active one.
	ShortCodeDeleteOverride = "override with newer key"
	// ShortCodeDeleteConsumed is the deletion comment set when the short code has been
	// redeemed successfully.
	ShortCodeDeleteConsumed = "key consumed"
)

// ShortCodeDeleteRequest is the input to [PgShortCodeDelete.Exec]. Comment is usually
// [ShortCodeDeleteOverride] or [ShortCodeDeleteConsumed]; any value is accepted and
// persisted on the deleted row for later auditing.
type ShortCodeDeleteRequest struct {
	// ID of the short code to delete.
	ID uuid.UUID
	// Now is the timestamp recorded as the row's deletion time.
	Now time.Time
	// Comment records the reason for the deletion. See [ShortCode.DeletedAt] for the lifecycle.
	Comment string
}

// PgShortCodeDelete soft-deletes an active short code, neither expired nor already deleted.
type PgShortCodeDelete struct{}

func NewPgShortCodeDelete() *PgShortCodeDelete {
	return &PgShortCodeDelete{}
}

func (dao *PgShortCodeDelete) Exec(ctx context.Context, request *ShortCodeDeleteRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgShortCodeDelete")
	defer span.End()

	span.SetAttributes(attribute.String("shortCode.comment", request.Comment))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &ShortCode{}

	err = tx.NewRaw(shortCodeDeleteQuery, request.Now, request.Comment, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrShortCodeDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
