package dao

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.authorizationCodeConsume.sql
var authorizationCodeConsumeQuery string

// ErrAuthorizationCodeConsumeNotFound is returned by [PgAuthorizationCodeConsume.Exec] when no
// unconsumed code has the requested ID, as when another exchange consumed it first.
var ErrAuthorizationCodeConsumeNotFound = errors.New("authorization code not found")

// AuthorizationCodeConsumeRequest is the input to [PgAuthorizationCodeConsume.Exec].
type AuthorizationCodeConsumeRequest struct {
	// See [AuthorizationCode.ID].
	ID []byte
	// Now is the instant recorded as [AuthorizationCode.ConsumedAt].
	Now time.Time
}

// PgAuthorizationCodeConsume marks an unconsumed authorization code consumed, in one conditional
// statement. Of several concurrent calls for one code, one alone succeeds.
type PgAuthorizationCodeConsume struct{}

func NewPgAuthorizationCodeConsume() *PgAuthorizationCodeConsume {
	return &PgAuthorizationCodeConsume{}
}

func (dao *PgAuthorizationCodeConsume) Exec(ctx context.Context, request *AuthorizationCodeConsumeRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgAuthorizationCodeConsume")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	res, err := db.NewRaw(authorizationCodeConsumeQuery, request.ID, request.Now).Exec(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	consumed, err := res.RowsAffected()
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	if consumed == 0 {
		return otel.ReportError(span, ErrAuthorizationCodeConsumeNotFound)
	}

	return nil
}
