package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
)

//go:embed select_credentials_by_email.sql
var selectCredentialsByEmailQuery string

// SelectCredentialsByEmailRepository is the repository used to perform the
// SelectCredentialsByEmailRepository.SelectCredentialsByEmail action.
//
// You may create one using the NewSelectCredentialsByEmailRepository function.
type SelectCredentialsByEmailRepository struct{}

func NewSelectCredentialsByEmailRepository() *SelectCredentialsByEmailRepository {
	return &SelectCredentialsByEmailRepository{}
}

// SelectCredentialsByEmail returns a set of credentials based on their email.
//
// The password is returned, encrypted, to allow for password verification (when creating a session). The result of
// this method MUST NEVER be sent to the client as-is, as it contains sensitive information.
//
// You may use SelectCredentialsRepository to retrieve credentials based on their unique identifier (ID).
func (repository *SelectCredentialsByEmailRepository) SelectCredentialsByEmail(
	ctx context.Context, email string,
) (*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ExistsCredentialsEmail")
	defer span.End()

	span.SetAttributes(attribute.String("email", email))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{}

	// Execute query.
	err = tx.NewRaw(selectCredentialsByEmailQuery, email).Scan(ctx, entity)
	if err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrCredentialsNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select key: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
