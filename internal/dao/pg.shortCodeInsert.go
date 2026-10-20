package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// ErrShortCodeInsertNested is returned when [ShortCodeInsert.Exec] runs inside a caller's
// transaction. The insert opens its own at read committed, and postgres.WithinTx ignores the options
// of a nested call.
var ErrShortCodeInsertNested = errors.New("short code insert cannot run inside another transaction")

//go:embed pg.shortCodeInsert.sql
var shortCodeInsertQuery string

// ErrShortCodeInsertAlreadyExists is returned by [ShortCodeInsert.Exec] when an active short code
// already covers the same (target, usage) pair. The conflict check of a request without Override
// finds it, and the partial unique index catches a concurrent insert.
var ErrShortCodeInsertAlreadyExists = errors.New("short code already exists")

// ShortCodeInsertRequest is the input to [ShortCodeInsert.Exec].
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

	// Override soft-deletes any conflicting record, with the [ShortCodeDeleteOverride] deletion
	// comment. When false, a conflict fails the insert.
	Override bool
}

// ShortCodeInsert inserts a new short code in the database.
type ShortCodeInsert struct{}

func NewShortCodeInsert() *ShortCodeInsert {
	return &ShortCodeInsert{}
}

func (dao *ShortCodeInsert) Exec(ctx context.Context, request *ShortCodeInsertRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("shortCode.id", request.ID.String()),
		attribute.String("shortCode.usage", request.Usage),
		attribute.String("shortCode.target", request.Target),
		attribute.Bool("shortCode.override", request.Override),
	)

	if postgres.InTx(ctx) {
		return nil, otel.ReportError(span, ErrShortCodeInsertNested)
	}

	entity := &ShortCode{}
	// The partial unique index keeps one non-deleted code per (target, usage) pair at any
	// isolation. Under read committed, the losing UPDATE of two concurrent overrides blocks and
	// re-evaluates, and its insert then fails on that index as an already-exists conflict.
	// Repeatable read fails it with a serialization error instead.
	txOptions := &sql.TxOptions{Isolation: sql.LevelReadCommitted}

	err := postgres.WithinTx(ctx, txOptions, func(ctx context.Context) error {
		var err error

		if request.Override {
			// Soft-delete every non-deleted row of the pair, expired or not, so the partial unique
			// index is clear before the insert.
			err = dao.discardConflicts(ctx, request)
		} else {
			// checkConflicts ignores expired rows, which still sit in the partial unique index
			// and collide with the insert. Soft-delete them first.
			err = dao.discardExpired(ctx, request)
			if err == nil {
				err = dao.checkConflicts(ctx, request)
			}
		}

		if err != nil {
			return fmt.Errorf("handle conflict: %w", err)
		}

		tx, err := postgres.GetContext(ctx)
		if err != nil {
			return fmt.Errorf("get database handle: %w", err)
		}

		err = tx.NewRaw(
			shortCodeInsertQuery,
			request.ID,
			request.Code,
			request.Usage,
			request.Target,
			request.Data,
			request.Now,
			request.ExpiresAt,
		).Scan(ctx, entity)
		if err != nil {
			// The partial unique index catches the conflicts the steps above miss under
			// concurrent inserts.
			if isUniqueViolation(err) {
				err = errors.Join(err, ErrShortCodeInsertAlreadyExists)
			}

			return fmt.Errorf("execute query: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return entity, nil
}

//go:embed pg.shortCodeInsert.discardConflict.sql
var shortCodeInsertDiscardConflictQuery string

func (dao *ShortCodeInsert) discardConflicts(ctx context.Context, request *ShortCodeInsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(discardConflicts)")
	defer span.End()

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get database handle: %w", err))
	}

	_, err = tx.NewRaw(
		shortCodeInsertDiscardConflictQuery,
		// Go and Postgres can disagree on "now" by a small margin, so backdate the
		// deletion by a second: the row then stays in the past for predicates
		// comparing deleted_at against CURRENT_TIMESTAMP, such as the active
		// short-codes view.
		lo.ToPtr(request.Now.Add(-time.Second)),
		lo.ToPtr(ShortCodeDeleteOverride),
		request.Target,
		request.Usage,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}

//go:embed pg.shortCodeInsert.discardExpired.sql
var shortCodeInsertDiscardExpiredQuery string

func (dao *ShortCodeInsert) discardExpired(ctx context.Context, request *ShortCodeInsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(discardExpired)")
	defer span.End()

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get database handle: %w", err))
	}

	_, err = tx.NewRaw(
		shortCodeInsertDiscardExpiredQuery,
		request.Now,
		"expired before insert",
		request.Target,
		request.Usage,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return nil
}

//go:embed pg.shortCodeInsert.checkConflict.sql
var shortCodeInsertCheckConflictQuery string

func (dao *ShortCodeInsert) checkConflicts(ctx context.Context, request *ShortCodeInsertRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(checkConflicts)")
	defer span.End()

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get database handle: %w", err))
	}

	res, err := tx.NewRaw(shortCodeInsertCheckConflictQuery, request.Target, request.Usage).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	if n == 1 {
		return otel.ReportError(span, ErrShortCodeInsertAlreadyExists)
	}

	return nil
}
