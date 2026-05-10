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
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.shortCodeInsert.sql
var shortCodeInsertQuery string

// ErrShortCodeInsertAlreadyExists is returned by [ShortCodeInsert.Exec] when an
// active short code already covers the same (target, usage) pair. Detection
// runs in two layers: an in-transaction conflict check on the Override=false
// path that short-circuits with this sentinel, and the database's partial
// unique index (target, usage) WHERE deleted_at IS NULL, which catches any
// race that slips past the check by mapping the resulting SQLSTATE 23505 onto
// the same sentinel. Callers branch on it with errors.Is regardless of which
// layer caught the conflict.
var ErrShortCodeInsertAlreadyExists = errors.New("short code already exists")

// ShortCodeInsertRequest is the input to [ShortCodeInsert.Exec]. The repository
// runs at REPEATABLE READ isolation; uniqueness on (target, usage) for the
// active subset is enforced by the partial unique index added in the
// 20260510140000 migration, so concurrent inserts cannot produce duplicates
// (the loser sees [ErrShortCodeInsertAlreadyExists]).
type ShortCodeInsertRequest struct {
	// See ShortCode.ID.
	ID uuid.UUID
	// See ShortCode.Code.
	Code string
	// See ShortCode.Usage.
	Usage string
	// See ShortCode.Target.
	Target string
	// See ShortCode.Data.
	Data []byte
	// Time used for insertion date.
	Now time.Time
	// See ShortCode.ExpiresAt.
	ExpiresAt time.Time

	// By default, the insertion will fail if the new short code does not meet the uniqueness
	// criteria. When this option is set to true, any conflicting record will be deleted
	// instead, with the ShortCodeDeleteOverride deletion comment.
	Override bool
}

// ShortCodeInsert inserts a new short code in the database.
type ShortCodeInsert struct{}

func NewShortCodeInsert() *ShortCodeInsert {
	return &ShortCodeInsert{}
}

func (repository *ShortCodeInsert) Exec(ctx context.Context, request *ShortCodeInsertRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert")
	defer span.End()

	span.SetAttributes(
		attribute.String("shortCode.id", request.ID.String()),
		attribute.String("shortCode.usage", request.Usage),
		attribute.String("shortCode.target", request.Target),
		attribute.Bool("shortCode.override", request.Override),
	)

	entity := new(ShortCode)
	txOptions := &sql.TxOptions{Isolation: sql.LevelRepeatableRead}

	err := postgres.RunInTx(ctx, txOptions, func(ctx context.Context, tx bun.IDB) error {
		// Soft-delete any naturally-expired-but-not-yet-deleted row first, on
		// both paths. Without this, the partial unique index
		// (target, usage) WHERE deleted_at IS NULL would block a fresh insert
		// against an expired stale row that discardConflicts (Override=true) or
		// checkConflicts (Override=false) would otherwise ignore — both queries
		// gate on `expires_at > now()`, so naturally-expired rows are invisible
		// to them but still live in the partial index.
		err := repository.discardExpired(ctx, tx, request)
		if err == nil {
			if request.Override {
				err = repository.discardConflicts(ctx, tx, request)
			} else {
				err = repository.checkConflicts(ctx, tx, request)
			}
		}

		if err != nil {
			return fmt.Errorf("handle conflict: %w", err)
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
			// The partial unique index (target, usage) WHERE deleted_at IS NULL
			// catches any conflict that the in-transaction check missed under
			// concurrent inserts. Map the SQLSTATE 23505 onto the same sentinel
			// callers already use so they don't need to distinguish the layer.
			var pgErr pgdriver.Error
			if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
				err = errors.Join(err, ErrShortCodeInsertAlreadyExists)
			}

			return fmt.Errorf("execute query: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}

//go:embed pg.shortCodeInsert.discardConflict.sql
var shortCodeInsertDiscardConflictQuery string

func (repository *ShortCodeInsert) discardConflicts(
	ctx context.Context, tx bun.IDB, request *ShortCodeInsertRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(discardConflicts)")
	defer span.End()

	_, err := tx.NewRaw(
		shortCodeInsertDiscardConflictQuery,
		// So as we switch between Go/Pg timestamps, and transactions / views / triggers,
		// there is a mismatch when comparing dates which can sometimes lead Postgres to
		// believe this (expired) row is still part of the active short codes view.
		//
		// To make sure any conflicting short code is considered expired by the time
		// we perform the insertion, we cheat and make the deletion date one second
		// older. This is usually enough to prevent any issue.
		lo.ToPtr(request.Now.Add(-time.Second)),
		lo.ToPtr(ShortCodeDeleteOverride),
		request.Target,
		request.Usage,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

//go:embed pg.shortCodeInsert.discardExpired.sql
var shortCodeInsertDiscardExpiredQuery string

func (repository *ShortCodeInsert) discardExpired(
	ctx context.Context, tx bun.IDB, request *ShortCodeInsertRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(discardExpired)")
	defer span.End()

	_, err := tx.NewRaw(
		shortCodeInsertDiscardExpiredQuery,
		request.Now,
		"expired before insert",
		request.Target,
		request.Usage,
	).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

//go:embed pg.shortCodeInsert.checkConflict.sql
var shortCodeInsertCheckConflictQuery string

func (repository *ShortCodeInsert) checkConflicts(
	ctx context.Context, tx bun.IDB, request *ShortCodeInsertRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeInsert(checkConflicts)")
	defer span.End()

	res, err := tx.NewRaw(shortCodeInsertCheckConflictQuery, request.Target, request.Usage).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	if n == 1 {
		return ErrShortCodeInsertAlreadyExists
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
