package dao

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

var ErrDeleteShortCodeRepository = errors.New("DeleteShortCodeRepository.DeleteShortCode")

func NewErrDeleteShortCodeRepository(err error) error {
	return errors.Join(err, ErrDeleteShortCodeRepository)
}

const (
	// DeleteCommentOverrideWithNewerKey is the default deletion comment set when a newer version of an active short
	// code is saved in the database.
	DeleteCommentOverrideWithNewerKey = "override with newer key"
	// DeleteCommentConsumed is the default deletion comment set when the resource the short code grants access to has
	// been consumed successfully.
	DeleteCommentConsumed = "key consumed"
)

// DeleteShortCodeData is the input used to perform the DeleteShortCodeRepository.DeleteShortCode action.
type DeleteShortCodeData struct {
	// ID of the short code to delete.
	ID uuid.UUID
	// Time at which the short code is marked as deleted.
	//
	// Once the date is reached, the short code is considered as expired and becomes invisible to the application.
	Now time.Time
	// Comment explaining the reason for the deletion.
	//
	// Available values for this field are:
	//  - DeleteCommentOverrideWithNewerKey
	//  - DeleteCommentConsumed
	Comment string
}

// DeleteShortCodeRepository is the repository used to perform the DeleteShortCodeRepository.DeleteShortCode action.
//
// You may create one using the NewDeleteShortCodeRepository function.
type DeleteShortCodeRepository struct{}

// DeleteShortCode performs a soft delete of a ShortCodeEntity.
//
// A short code is a one-time password used to grant a user access to a specific resource. Only one instance of
// a short code must exist for a given target and resource at a given time.
//
// When a short code is used, or an external action makes it obsolete, it must be deleted to prevent further use.
// The short code is only soft-deleted, so admins may have access to it when needed for investigation.
//
// A comment gives more insights about the reason a short code was deleted. This comment is required, and under
// normal circumstances, MUST be a static, generic value to facilitate the understanding of the deletion.
//
// This method also returns an error when the short code is not found, so you can be sure something was deleted on
// success.
// The deleted short code is returned on success.
func (repository *DeleteShortCodeRepository) DeleteShortCode(
	ctx context.Context, data DeleteShortCodeData,
) (*ShortCodeEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, NewErrDeleteShortCodeRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &ShortCodeEntity{
		ID:             data.ID,
		DeletedAt:      &data.Now,
		DeletedComment: &data.Comment,
	}

	// Execute query.
	res, err := tx.NewUpdate().
		Model(entity).
		Where("id = ?", data.ID).
		Column("deleted_at", "deleted_comment"). // Only update the deletion fields.
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, NewErrDeleteShortCodeRepository(fmt.Errorf("delete short code: %w", err))
	}

	// Ensure something has been deleted.
	// This operation should never fail, as we use a driver that supports it.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, NewErrDeleteShortCodeRepository(fmt.Errorf("get rows affected: %w", err))
	}

	if rowsAffected == 0 {
		return nil, NewErrDeleteShortCodeRepository(fmt.Errorf("delete key: %w", ErrShortCodeNotFound))
	}

	return entity, nil
}

func NewDeleteShortCodeRepository() *DeleteShortCodeRepository {
	return &DeleteShortCodeRepository{}
}
