package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed pg.credentialsExist.sql
var credentialsExistQQuery string

type CredentialsExistRequest struct {
	Email string
}

type CredentialsExist struct{}

func NewCredentialsExist() *CredentialsExist {
	return new(CredentialsExist)
}

func (repository *CredentialsExist) Exec(ctx context.Context, request *CredentialsExistRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsExist")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return false, fmt.Errorf("get transaction: %w", otel.ReportError(span, err))
	}

	res, err := tx.NewRaw(credentialsExistQQuery, request.Email).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return otel.ReportSuccess(span, n >= 1), nil
}
