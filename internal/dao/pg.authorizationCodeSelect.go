package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.authorizationCodeSelect.sql
var authorizationCodeSelectQuery string

// AuthorizationCodeSelectRequest is the input to [PgAuthorizationCodeSelect.Exec].
type AuthorizationCodeSelectRequest struct {
	// See [AuthorizationCode.ID].
	ID []byte
}

// PgAuthorizationCodeSelect fetches an authorization code by ID, consumed or expired ones included.
type PgAuthorizationCodeSelect struct{}

func NewPgAuthorizationCodeSelect() *PgAuthorizationCodeSelect {
	return &PgAuthorizationCodeSelect{}
}

func (dao *PgAuthorizationCodeSelect) Exec(
	ctx context.Context, request *AuthorizationCodeSelectRequest,
) (*AuthorizationCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgAuthorizationCodeSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &AuthorizationCode{}

	err = db.NewRaw(authorizationCodeSelectQuery, request.ID).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrAuthorizationCodeNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
