package dao

import (
	"context"
	"database/sql"
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
// statement, and returns it. Of several concurrent calls for one code, one alone succeeds.
type PgAuthorizationCodeConsume struct{}

func NewPgAuthorizationCodeConsume() *PgAuthorizationCodeConsume {
	return &PgAuthorizationCodeConsume{}
}

func (dao *PgAuthorizationCodeConsume) Exec(
	ctx context.Context, request *AuthorizationCodeConsumeRequest,
) (*AuthorizationCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgAuthorizationCodeConsume")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &AuthorizationCode{}

	err = db.NewRaw(authorizationCodeConsumeQuery, request.ID, request.Now).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrAuthorizationCodeConsumeNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
