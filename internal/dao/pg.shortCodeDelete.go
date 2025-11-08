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

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.shortCodeDelete.sql
var shortCodeDeleteQuery string

var ErrShortCodeDeleteNotFound = errors.New("short code not found")

const (
	// ShortCodeDeleteOverride is the default deletion comment set when a newer version of an active short
	// code is saved in the database.
	ShortCodeDeleteOverride = "override with newer key"
	// ShortCodeDeleteConsumed is the default deletion comment set when the resource the short code grants access to has
	// been consumed successfully.
	ShortCodeDeleteConsumed = "key consumed"
)

type ShortCodeDeleteRequest struct {
	ID      uuid.UUID
	Now     time.Time
	Comment string
}

type ShortCodeDelete struct{}

func NewShortCodeDelete() *ShortCodeDelete {
	return new(ShortCodeDelete)
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
			return nil, otel.ReportError(span, errors.Join(err, ErrShortCodeDeleteNotFound))
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
