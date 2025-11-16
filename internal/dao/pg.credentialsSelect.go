package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsSelect.sql
var credentialsSelectQuery string

var ErrCredentialsSelectNotFound = errors.New("credentials not found")

type CredentialsSelectRequest struct {
	ID uuid.UUID
}

type CredentialsSelect struct{}

func NewCredentialsSelect() *CredentialsSelect {
	return new(CredentialsSelect)
}

func (repository *CredentialsSelect) Exec(
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
