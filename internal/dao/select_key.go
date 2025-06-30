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

var ErrSelectKeyRepository = errors.New("SelectKeyRepository.SelectKey")

func NewErrSelectKeyRepository(err error) error {
	return errors.Join(err, ErrSelectKeyRepository)
}

// SelectKeyRepository is the repository used to perform the SelectKeyRepository.SelectKey action.
//
// You may create one using the NewSelectKeyRepository function.
type SelectKeyRepository struct{}

func NewSelectKeyRepository() *SelectKeyRepository {
	return &SelectKeyRepository{}
}

// SelectKey returns a public/private key pair based on their unique identifier (ID).
//
// The ID of a key pair is usually carried by the payload they were used on, for example thw KIS field of a JWT header.
// This allows to retrieve the exact key when performing reverse operations (signature verification or token
// decryption).
func (repository *SelectKeyRepository) SelectKey(ctx context.Context, id uuid.UUID) (*KeyEntity, error) {
	span := sentry.StartSpan(ctx, "SelectKeyRepository.SelectKey")
	defer span.Finish()

	span.SetData("key.id", id.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrSelectKeyRepository(fmt.Errorf("get postgres client: %w", err))
	}

	var entity KeyEntity

	// Execute query.
	err = tx.NewSelect().Model(&entity).Where("id = ?", id).Order("id DESC").Scan(span.Context())
	if err != nil {
		span.SetData("scan.error", err.Error())

		// Parse not found error as a managed error.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewErrSelectKeyRepository(ErrKeyNotFound)
		}

		return nil, NewErrSelectKeyRepository(fmt.Errorf("select key: %w", err))
	}

	return &entity, nil
}
