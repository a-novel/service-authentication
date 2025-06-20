package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/a-novel/service-authentication/internal/lib"

	"github.com/google/uuid"
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
	tx, err := lib.PostgresContext(ctx)
	if err != nil {
		return nil, NewErrSelectCredentialsRepository(fmt.Errorf("get postgres client: %w", err))
	}

	var entity CredentialsEntity

	// Execute query.
	if err = tx.NewSelect().Model(&entity).Where("id = ?", id).Order("id DESC").Scan(ctx); err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewErrSelectCredentialsRepository(ErrCredentialsNotFound)
		}

		return nil, NewErrSelectCredentialsRepository(fmt.Errorf("select key: %w", err))
	}

	return &entity, nil
}

func NewSelectCredentialsRepository() *SelectCredentialsRepository {
	return &SelectCredentialsRepository{}
}
