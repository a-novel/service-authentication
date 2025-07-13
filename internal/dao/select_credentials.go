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

//go:embed select_credentials.sql
var selectCredentialsQuery string

// SelectCredentialsRepository is the repository used to perform the SelectCredentialsRepository.SelectCredentials
// action.
//
// You may create one using the NewSelectCredentialsRepository function.
type SelectCredentialsRepository struct{}

func NewSelectCredentialsRepository() *SelectCredentialsRepository {
	return &SelectCredentialsRepository{}
}

// SelectCredentials returns a set of credentials based on their unique identifier (ID).
//
// The password is returned, encrypted, to allow for password verification (when creating a session). The result of
// this method MUST NEVER be sent to the client as-is, as it contains sensitive information.
//
// You may use SelectCredentialsByEmailRepository to retrieve credentials based on their email.
func (repository *SelectCredentialsRepository) SelectCredentials(
	ctx context.Context, id uuid.UUID,
) (*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SelectCredentials")
	defer span.End()

	span.SetAttributes(attribute.String("id", id.String()))

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{}

	err = tx.NewRaw(selectCredentialsQuery, id).Scan(ctx, entity)
	if err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrCredentialsNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select key: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
