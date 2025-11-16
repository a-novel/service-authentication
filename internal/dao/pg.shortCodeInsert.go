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
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.shortCodeInsert.sql
var shortCodeInsertQuery string

var ErrShortCodeInsertAlreadyExists = errors.New("short code already exists")

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
	return new(ShortCodeInsert)
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
		var err error

		if request.Override {
			err = repository.discardConflicts(ctx, tx, request)
		} else {
			err = repository.checkConflicts(ctx, tx, request)
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
		return otel.ReportError(span, ErrShortCodeInsertAlreadyExists)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
