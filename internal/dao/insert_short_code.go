package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/models"
)

//go:embed insert_short_code.sql
var insertShortCodeQuery string

// InsertShortCodeData is the input used to perform the InsertShortCodeRepository.InsertShortCode action.
type InsertShortCodeData struct {
	// ID of the short code. It must be unique (random).
	ID uuid.UUID

	// The encrypted code. A clear version of this code is sent to the target.
	Code string
	// Information about the resource the short code grants access to.
	Usage models.ShortCodeUsage
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string
	// Data used for the targeted resource. It can contain any information required to perform a specific action.
	Data []byte

	// Time at which the short code was created.
	Now time.Time
	// Expiration of the short code. Each short code is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the short code can no longer be used or retrieved.
	ExpiresAt time.Time

	// Whe true, automatically expires any existing short code with the same target and usage.
	// Otherwise, the presence of a duplicate will trigger an error.
	//
	// Deleted duplicates will have their deletion comment set to DeleteCommentOverrideWithNewerKey.
	Override bool
}

// InsertShortCodeRepository is the repository used to perform the InsertShortCodeRepository.InsertShortCode action.
//
// You may create one using the NewInsertShortCodeRepository function.
type InsertShortCodeRepository struct{}

func NewInsertShortCodeRepository() *InsertShortCodeRepository {
	return &InsertShortCodeRepository{}
}

// InsertShortCode inserts a new short code in the database.
//
// The clear value of the short code MUST not be saved with this repository, but instead sent to the target using a
// secure channel. The encrypted version of the short code is stored in the database.
//
// A short code is uniquely identified by its target and usage. If another code with the same target and usage is
// found, the insertion will fail. You can override this behavior, and use the InsertShortCodeData.Override field to
// automatically expire any conflicting short code.
func (repository *InsertShortCodeRepository) InsertShortCode(
	ctx context.Context, data InsertShortCodeData,
) (*ShortCodeEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.InsertShortCodeRepository")
	defer span.End()

	span.SetAttributes(
		attribute.String("shortCode.id", data.ID.String()),
		attribute.String("shortCode.usage", data.Usage.String()),
		attribute.String("shortCode.target", data.Target),
		attribute.Bool("shortCode.override", data.Override),
	)

	output := &ShortCodeEntity{}

	txOptions := &sql.TxOptions{Isolation: sql.LevelRepeatableRead}

	err := postgres.RunInTx(ctx, txOptions, func(ctx context.Context, tx bun.IDB) error {
		var err error

		if data.Override {
			err = repository.discardConflicts(ctx, tx, data)
		} else {
			err = repository.checkConflicts(ctx, tx, data)
		}

		if err != nil {
			return fmt.Errorf("insert short code: %w", err)
		}

		return tx.NewRaw(
			insertShortCodeQuery,
			data.ID,
			data.Code,
			data.Usage,
			data.Target,
			data.Data,
			data.Now,
			data.ExpiresAt,
		).Scan(ctx, output)
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return otel.ReportSuccess(span, output), nil
}

//go:embed insert_short_code.discard_conflict.sql
var discardShortCodeConflictQuery string

func (repository *InsertShortCodeRepository) discardConflicts(
	ctx context.Context, tx bun.IDB, data InsertShortCodeData,
) error {
	// Discard any conflicting short codes before starting.
	_, err := tx.NewRaw(
		discardShortCodeConflictQuery,
		// So as we switch between Go/Pg timestamps, and transactions / views / triggers,
		// there is a mismatch when comparing dates which can sometimes lead Postgres to
		// believe this (expired) row is still part of the active short codes view.
		//
		// To make sure any conflicting short code is considered expired by the time
		// we perform the insertion, we cheat and make the deletion date one second
		// older. This is usually enough to prevent any issue.
		lo.ToPtr(data.Now.Add(-time.Second)),
		lo.ToPtr(DeleteCommentOverrideWithNewerKey),
		data.Target,
		data.Usage,
	).Exec(ctx)

	return err
}

//go:embed insert_short_code.check_conflict.sql
var checkShortCodeConflictQuery string

func (repository *InsertShortCodeRepository) checkConflicts(
	ctx context.Context, tx bun.IDB, data InsertShortCodeData,
) error {
	res, err := tx.NewRaw(checkShortCodeConflictQuery, data.Target, data.Usage).Exec(ctx)
	if err != nil {
		return fmt.Errorf("check short code existence: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if n == 1 {
		return ErrShortCodeAlreadyExists
	}

	return nil
}
