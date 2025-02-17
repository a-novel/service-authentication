package dao

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

// SelectKeyRepository is the repository used to perform the SelectKeyRepository.SelectKey action.
//
// You may create one using the NewSelectKeyRepository function.
type SelectKeyRepository struct{}

// SelectKey returns a public/private key pair based on their unique identifier (ID).
//
// The ID of a key pair is usually carried by the payload they were used on, for example thw KIS field of a JWT header.
// This allows to retrieve the exact key when performing reverse operations (signature verification or token
// decryption).
func (repository *SelectKeyRepository) SelectKey(ctx context.Context, id uuid.UUID) (*KeyEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, fmt.Errorf("(SelectKeyRepository.SelectKey) get postgres client: %w", err)
	}

	var entity KeyEntity

	// Execute query.
	if err = tx.NewSelect().Model(&entity).Where("id = ?", id).Scan(ctx); err != nil {
		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("(SelectKeyRepository.SelectKey) select key: %w", ErrKeyNotFound)
		}

		return nil, fmt.Errorf("(SelectKeyRepository.SelectKey) select key: %w", err)
	}

	return &entity, nil
}

func NewSelectKeyRepository() *SelectKeyRepository {
	return &SelectKeyRepository{}
}
