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

var ErrShortCodeDeleteNotFound = errors.New("short code not found")

const (
	// ShortCodeDeleteOverride is the deletion comment set when a newer version of an active short
	// code is saved in the database.
	ShortCodeDeleteOverride = "override with newer key"
	// ShortCodeDeleteConsumed is the deletion comment set when the resource the short code grants access to has
	// been consumed successfully.
	ShortCodeDeleteConsumed = "key consumed"
)

// ShortCodeDeleteRequest is the input to [ShortCodeDelete.Exec]. Comment should
// usually be one of the [ShortCodeDeleteOverride] / [ShortCodeDeleteConsumed]
// constants, but any value is accepted and persisted on the deleted row for
// later auditing.
type ShortCodeDeleteRequest struct {
	// ID of the short code to delete.
	ID uuid.UUID
	// Now is the timestamp recorded as the row's deletion time.
	Now time.Time
	// Comment records the reason for the deletion; persisted on the row's
	// deleted_comment column. See [ShortCode.DeletedAt] for the lifecycle.
	Comment string
}

// ShortCodeDelete deletes a non-expired short code.
type ShortCodeDelete struct{}

func NewShortCodeDelete() *ShortCodeDelete {
	return &ShortCodeDelete{}
}

func (repository *ShortCodeDelete) Exec(ctx context.Context, request *ShortCodeDeleteRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("shortCode.id", request.ID.String()),
		attribute.Int64("shortCode.now", request.Now.Unix()),
		attribute.String("shortCode.comment", request.Comment),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(ShortCode)

	err = tx.NewRaw(shortCodeDeleteQuery, request.Now, request.Comment, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrShortCodeDeleteNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
