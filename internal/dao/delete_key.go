package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/getsentry/sentry-go"
	"time"

	"github.com/google/uuid"
)

var ErrDeleteKeyRepository = errors.New("DeleteKeyRepository.DeleteKey")

func NewErrDeleteKeyRepository(err error) error {
	return errors.Join(err, ErrDeleteKeyRepository)
}

// DeleteKeyData is the input used to perform the DeleteKeyRepository.DeleteKey action.
type DeleteKeyData struct {
	// ID of the key to delete.
	ID uuid.UUID
	// Time at which the key is marked as deleted. This time might be set in the near future to delay the deletion.
	//
	// Once the date is reached, the key is considered as expired and becomes invisible to the application.
	Now time.Time
	// Comment explaining the circumstances surrounding the deletion of the key.
	Comment string
}

// DeleteKeyRepository is the repository used to perform the DeleteKeyRepository.DeleteKey action.
//
// You may create one using the NewDeleteKeyRepository function.
type DeleteKeyRepository struct{}

// DeleteKey performs a soft delete of a KeyEntity.
//
// A KeyEntity expires naturally through its KeyEntity.ExpiresAt field. However, some circumstances may require a key
// to be invalidated earlier (e.g. a security breach). In such cases, this method can be used.
//
// Once a key is marked as deleted, it is not removed from the database to allow further investigation. It is simply
// removed from the main view, which means the application will not see it anymore.
//
// As this method is not intended to be used "normally", a comment giving more details about the circumstance
// surrounding the deletion is required.
//
// This method also returns an error when the key is not found, so you can be sure something was deleted on success.
// The deleted key is returned on success.
func (repository *DeleteKeyRepository) DeleteKey(ctx context.Context, data DeleteKeyData) (*KeyEntity, error) {
	span := sentry.StartSpan(ctx, "DeleteKeyRepository.DeleteKey")
	defer span.Finish()

	span.SetData("key.id", data.ID.String())
	span.SetData("key.now", data.Now.String())
	span.SetData("key.comment", data.Comment)

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrDeleteKeyRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &KeyEntity{
		ID:             data.ID,
		DeletedAt:      &data.Now,
		DeletedComment: &data.Comment,
	}

	// Execute query.
	res, err := tx.NewUpdate().
		Model(entity).
		ModelTableExpr("active_keys").
		Where("id = ?", data.ID).
		Column("deleted_at", "deleted_comment"). // Only update the deletion-related fields.
		Returning("*").
		Exec(span.Context())
	if err != nil {
		span.SetData("update.error", err.Error())

		return nil, NewErrDeleteKeyRepository(fmt.Errorf("delete key: %w", err))
	}

	// Ensure something has been deleted.
	// This operation should never fail, as we use a driver that supports it.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.SetData("rowsAffected.error", err.Error())

		return nil, NewErrDeleteKeyRepository(fmt.Errorf("delete key: %w", err))
	}

	span.SetData("rowsAffected", rowsAffected)

	if rowsAffected == 0 {
		span.SetData("error", "key not found")

		return nil, NewErrDeleteKeyRepository(fmt.Errorf("delete key: %w", ErrKeyNotFound))
	}

	return entity, nil
}

func NewDeleteKeyRepository() *DeleteKeyRepository {
	return &DeleteKeyRepository{}
}
