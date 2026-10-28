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

//go:embed pg.shortCodeRenew.sql
var shortCodeRenewQuery string

// ErrShortCodeRenewNotFound is returned by [PgShortCodeRenew.Exec] when the short code is deleted:
// consumed, superseded by a newer one, or suppressed.
var ErrShortCodeRenewNotFound = errors.New("short code not found")

// ShortCodeRenewRequest is the input to [PgShortCodeRenew.Exec].
type ShortCodeRenewRequest struct {
	// ID identifies the short code to renew.
	ID uuid.UUID
	// Code is the hash of the new code. See [ShortCode.Code].
	Code string
	// Now is the new creation time.
	Now time.Time
	// ExpiresAt is the new expiry. See [ShortCode.ExpiresAt].
	ExpiresAt time.Time
}

// PgShortCodeRenew replaces the code of a short code that is not deleted, which invalidates the
// previous code. An expired short code is renewed too.
type PgShortCodeRenew struct{}

func NewPgShortCodeRenew() *PgShortCodeRenew {
	return &PgShortCodeRenew{}
}

func (dao *PgShortCodeRenew) Exec(ctx context.Context, request *ShortCodeRenewRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgShortCodeRenew")
	defer span.End()

	span.SetAttributes(attribute.String("short_code.id", request.ID.String()))

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &ShortCode{}

	err = db.NewRaw(shortCodeRenewQuery, request.ID, request.Code, request.Now, request.ExpiresAt).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrShortCodeRenewNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
