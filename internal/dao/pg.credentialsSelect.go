package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsSelect.sql
var credentialsSelectQuery string

// ErrCredentialsSelectNotFound is returned by [CredentialsSelect.Exec] when no
// row matches the requested ID. It is joined onto the underlying sql.ErrNoRows
// so callers can branch on it with errors.Is.
var ErrCredentialsSelectNotFound = errors.New("credentials not found")

// CredentialsSelectRequest is the input to [CredentialsSelect.Exec].
type CredentialsSelectRequest struct {
	// ID of the credentials to fetch.
	ID uuid.UUID
}

// CredentialsSelect fetches a single credentials row by ID. Use
// [CredentialsSelectByEmail] to look up by email instead.
type CredentialsSelect struct{}

func NewCredentialsSelect() *CredentialsSelect {
	return &CredentialsSelect{}
}

func (dao *CredentialsSelect) Exec(
	ctx context.Context, request *CredentialsSelectRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsSelect")
	defer span.End()

	span.SetAttributes(attribute.String("id", request.ID.String()))

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(Credentials)

	err = tx.NewRaw(credentialsSelectQuery, request.ID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrCredentialsSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
