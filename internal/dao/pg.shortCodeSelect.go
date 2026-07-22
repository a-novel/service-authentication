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

// ErrShortCodeSelectNotFound is returned by [ShortCodeSelect.Exec] when no active
// short code matches the (Usage, Target) pair. The query filters expired and deleted
// rows in its WHERE clause (deleted_at IS NULL AND expires_at > CURRENT_TIMESTAMP),
// so the sentinel covers every state the DAO cannot tell apart: never issued, expired,
// and already consumed.
var ErrShortCodeSelectNotFound = errors.New("short code not found")

// ShortCodeSelectRequest is the input to [ShortCodeSelect.Exec]. A partial unique
// index keeps at most one active row per (Usage, Target) pair, so the query returns
// either zero or one row.
type ShortCodeSelectRequest struct {
	// Usage selects which flow this code is valid for; matches [ShortCode.Usage].
	Usage string
	// Target identifies the subject of the operation (e.g. the email address);
	// matches [ShortCode.Target].
	Target string
}

// ShortCodeSelect fetches an active short code matching a (Usage, Target) pair,
// if any.
type ShortCodeSelect struct{}

func NewShortCodeSelect() *ShortCodeSelect {
	return &ShortCodeSelect{}
}

func (dao *ShortCodeSelect) Exec(ctx context.Context, request *ShortCodeSelectRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("data", request.Target),
		attribute.String("usage", request.Usage),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(ShortCode)

	err = tx.NewRaw(shortCodeSelectQuery, request.Target, request.Usage).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrShortCodeSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
