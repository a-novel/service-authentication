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

// ErrShortCodeInsertNested is returned when [PgShortCodeInsert.Exec] runs inside a caller's
// transaction. The insert opens its own at read committed, and postgres.WithinTx ignores the options
// of a nested call.
var ErrShortCodeInsertNested = errors.New("short code insert cannot run inside another transaction")

//go:embed pg.shortCodeInsert.sql
var shortCodeInsertQuery string

//go:embed pg.shortCodeInsert.discardConflict.sql
var shortCodeInsertDiscardConflictQuery string

// ShortCodeInsertRequest is the input to [PgShortCodeInsert.Exec].
type ShortCodeInsertRequest struct {
	// See [ShortCode.ID].
	ID uuid.UUID
	// See [ShortCode.Code].
	Code string
	// See [ShortCode.Usage].
	Usage string
	// See [ShortCode.Target].
	Target string
	// See [ShortCode.Data].
	Data []byte
	// Now is the timestamp recorded as the row's creation time.
	Now time.Time
	// See [ShortCode.ExpiresAt].
	ExpiresAt time.Time
}

// PgShortCodeInsert replaces the short code of a (target, usage) pair: it soft-deletes the pair's
// live codes, with the [ShortCodeDeleteOverride] comment, then inserts the new one.
type PgShortCodeInsert struct{}

func NewPgShortCodeInsert() *PgShortCodeInsert {
	return &PgShortCodeInsert{}
}

func (dao *PgShortCodeInsert) Exec(ctx context.Context, request *ShortCodeInsertRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgShortCodeInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("short_code.id", request.ID.String()),
		attribute.String("short_code.usage", request.Usage),
		attribute.String("short_code.target", request.Target),
	)

	if postgres.InTx(ctx) {
		return nil, otel.ReportError(span, ErrShortCodeInsertNested)
	}

	entity := &ShortCode{}
	// The partial unique index keeps one non-deleted code per (target, usage) pair at any
	// isolation. Under read committed, the losing UPDATE of two concurrent inserts blocks and
	// re-evaluates, and its insert then fails on that index as an already-exists conflict.
	// Repeatable read fails it with a serialization error instead.
	txOptions := &sql.TxOptions{Isolation: sql.LevelReadCommitted}

	err := postgres.WithinTx(ctx, txOptions, func(ctx context.Context) error {
		db, err := postgres.GetContext(ctx)
		if err != nil {
			return fmt.Errorf("get postgres context: %w", err)
		}

		// Soft-delete every non-deleted code of the pair, expired or not, so the partial unique index
		// is clear before the insert. Go and Postgres can disagree on "now" by a small margin, so the
		// deletion is backdated by a second: the row then stays in the past for predicates comparing
		// deleted_at against CURRENT_TIMESTAMP, such as the active short-codes view.
		_, err = db.NewRaw(
			shortCodeInsertDiscardConflictQuery,
			request.Now.Add(-time.Second),
			ShortCodeDeleteOverride,
			request.Target,
			request.Usage,
		).Exec(ctx)
		if err != nil {
			return fmt.Errorf("discard live codes: %w", err)
		}

		err = db.NewRaw(
			shortCodeInsertQuery,
			request.ID,
			request.Code,
			request.Usage,
			request.Target,
			request.Data,
			request.Now,
			request.ExpiresAt,
		).Scan(ctx, entity)
		if code, _ := postgres.SQLState(err); code == pgUniqueViolation {
			err = errors.Join(err, ErrShortCodeAlreadyExists)
		}

		if err != nil {
			return fmt.Errorf("execute query: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return entity, nil
}
