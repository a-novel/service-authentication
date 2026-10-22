package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.shortCodeSelect.sql
var shortCodeSelectQuery string

// ErrShortCodeSelectNotFound is returned by [PgShortCodeSelect.Exec] when no active short code matches
// the (Usage, Target) pair. It covers a code never issued, expired and already consumed alike.
var ErrShortCodeSelectNotFound = errors.New("short code not found")

// ShortCodeSelectRequest is the input to [PgShortCodeSelect.Exec]. A partial unique index keeps at
// most one active row per (Usage, Target) pair.
type ShortCodeSelectRequest struct {
	// Usage selects which flow this code is valid for; matches [ShortCode.Usage].
	Usage string
	// Target identifies the subject of the operation; matches [ShortCode.Target].
	Target string
}

// PgShortCodeSelect fetches the active short code of a (Usage, Target) pair.
type PgShortCodeSelect struct{}

func NewPgShortCodeSelect() *PgShortCodeSelect {
	return &PgShortCodeSelect{}
}

func (dao *PgShortCodeSelect) Exec(ctx context.Context, request *ShortCodeSelectRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgShortCodeSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("short_code.target", request.Target),
		attribute.String("short_code.usage", request.Usage),
	)

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ShortCode{}

	err = db.NewRaw(shortCodeSelectQuery, request.Target, request.Usage).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrShortCodeSelectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
