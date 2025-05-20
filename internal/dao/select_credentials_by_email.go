package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	pgctx "github.com/a-novel-kit/context/pgbun"
)

var ErrSelectCredentialsByEmailRepository = errors.New("SelectCredentialsByEmailRepository.SelectCredentialsByEmail")

func NewErrSelectCredentialsByEmailRepository(err error) error {
	return errors.Join(err, ErrSelectCredentialsByEmailRepository)
}

// SelectCredentialsByEmailRepository is the repository used to perform the
// SelectCredentialsByEmailRepository.SelectCredentialsByEmail action.
//
// You may create one using the NewSelectCredentialsByEmailRepository function.
type SelectCredentialsByEmailRepository struct{}

// SelectCredentialsByEmail returns a set of credentials based on their email.
//
// The password is returned, encrypted, to allow for password verification (when creating a session). The result of
// this method MUST NEVER be sent to the client as-is, as it contains sensitive information.
//
// You may use SelectCredentialsRepository to retrieve credentials based on their unique identifier (ID).
func (repository *SelectCredentialsByEmailRepository) SelectCredentialsByEmail(
	ctx context.Context, email string,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, NewErrSelectCredentialsByEmailRepository(fmt.Errorf("get postgres client: %w", err))
	}

	var entity CredentialsEntity

	// Execute query.
	if err = tx.NewSelect().Model(&entity).Where("email = ?", email).Order("email DESC").Scan(ctx); err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewErrSelectCredentialsByEmailRepository(ErrCredentialsNotFound)
		}

		return nil, NewErrSelectCredentialsByEmailRepository(fmt.Errorf("select key: %w", err))
	}

	return &entity, nil
}

func NewSelectCredentialsByEmailRepository() *SelectCredentialsByEmailRepository {
	return &SelectCredentialsByEmailRepository{}
}
