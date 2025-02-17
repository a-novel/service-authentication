package dao

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

// SelectCredentialsRepository is the repository used to perform the SelectCredentialsRepository.SelectCredentials
// action.
//
// You may create one using the NewSelectCredentialsRepository function.
type SelectCredentialsRepository struct{}

// SelectCredentials returns a set of credentials based on their unique identifier (ID).
//
// The password is returned, encrypted, to allow for password verification (when creating a session). The result of
// this method MUST NEVER be sent to the client as-is, as it contains sensitive information.
//
// You may use SelectCredentialsByEmailRepository to retrieve credentials based on their email.
func (repository *SelectCredentialsRepository) SelectCredentials(
	ctx context.Context, id uuid.UUID,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, fmt.Errorf("(SelectCredentialsRepository.SelectCredentials) get postgres client: %w", err)
	}

	var entity CredentialsEntity

	// Execute query.
	if err = tx.NewSelect().Model(&entity).Where("id = ?", id).Scan(ctx); err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("(SelectCredentialsRepository.SelectCredentials): %w", ErrCredentialsNotFound)
		}

		return nil, fmt.Errorf("(SelectCredentialsRepository.SelectCredentials) select key: %w", err)
	}

	return &entity, nil
}

func NewSelectCredentialsRepository() *SelectCredentialsRepository {
	return &SelectCredentialsRepository{}
}
