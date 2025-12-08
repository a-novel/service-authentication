package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.credentialsExist.sql
var credentialsExistQQuery string

type CredentialsExistRequest struct {
	Email string
}

// CredentialsExist checks whether a user with a given email address exists or not.
//
// It does not return an error if the user does not exist, instead it returns a false boolean.
type CredentialsExist struct{}

func NewCredentialsExist() *CredentialsExist {
	return new(CredentialsExist)
}

func (repository *CredentialsExist) Exec(ctx context.Context, request *CredentialsExistRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.CredentialsExist")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

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
