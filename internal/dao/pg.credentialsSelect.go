package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsSelect.sql
var credentialsSelectQuery string

// ErrCredentialsSelectNotFound is returned by [PgCredentialsSelect.Exec] when no row matches the
// requested ID.
var ErrCredentialsSelectNotFound = errors.New("credentials not found")

// CredentialsSelectRequest is the input to [PgCredentialsSelect.Exec].
type CredentialsSelectRequest struct {
	// ID of the credentials to fetch.
	ID uuid.UUID
}

// PgCredentialsSelect fetches a single credentials row by ID. [PgCredentialsSelectByEmail] looks one up
// by email.
type PgCredentialsSelect struct{}

func NewPgCredentialsSelect() *PgCredentialsSelect {
	return &PgCredentialsSelect{}
}

func (dao *PgCredentialsSelect) Exec(
	ctx context.Context, request *CredentialsSelectRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgCredentialsSelect")
	defer span.End()

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &Credentials{}

	err = tx.NewRaw(credentialsSelectQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
