package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed exists_credentials_email.sql
var existsCredentialsEmailQuery string

// ExistsCredentialsEmailRepository is the repository used to perform the
// ExistsCredentialsEmailRepository.ExistsCredentialsEmail action.
//
// You may create one using the NewExistsCredentialsEmailRepository function.
type ExistsCredentialsEmailRepository struct{}

func NewExistsCredentialsEmailRepository() *ExistsCredentialsEmailRepository {
	return &ExistsCredentialsEmailRepository{}
}

// ExistsCredentialsEmail returns whether a credential with the given email exists in the database.
//
// Emails MUST be unique, and this method should be used to verify the availability of an address before creating it.
func (repository *ExistsCredentialsEmailRepository) ExistsCredentialsEmail(
	ctx context.Context, email string,
) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ExistsCredentialsEmail")
	defer span.End()

	span.SetAttributes(attribute.String("email", email))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	res, err := tx.NewRaw(existsCredentialsEmailQuery, email).Exec(ctx)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("check database: %w", err))
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("get rows affected: %w", err))
	}

	return otel.ReportSuccess(span, n == 1), nil
}
