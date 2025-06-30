package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/lib"
)

var ErrSelectCredentialsRepository = errors.New("SelectCredentialsRepository.SelectCredentials")

func NewErrSelectCredentialsRepository(err error) error {
	return errors.Join(err, ErrSelectCredentialsRepository)
}

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
	span := sentry.StartSpan(ctx, "SelectCredentialsRepository.SelectCredentials")
	defer span.Finish()

	span.SetData("credentials.id", id.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrSelectCredentialsRepository(fmt.Errorf("get postgres client: %w", err))
	}

	var entity CredentialsEntity

	// Execute query.
	err = tx.NewSelect().Model(&entity).Where("id = ?", id).Order("id DESC").Scan(span.Context())
	if err != nil {
		span.SetData("scan.error", err.Error())

		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewErrSelectCredentialsRepository(ErrCredentialsNotFound)
		}

		return nil, NewErrSelectCredentialsRepository(fmt.Errorf("select key: %w", err))
	}

	return &entity, nil
}
