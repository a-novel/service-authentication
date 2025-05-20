package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"

	pgctx "github.com/a-novel-kit/context/pgbun"
)

var ErrUpdateCredentialsEmailRepository = errors.New("UpdateCredentialsEmailRepository.UpdateCredentialsEmail")

func NewErrUpdateCredentialsEmailRepository(err error) error {
	return errors.Join(err, ErrUpdateCredentialsEmailRepository)
}

// UpdateCredentialsEmailData is the input used to perform the
// UpdateCredentialsEmailRepository.UpdateCredentialsEmail action.
type UpdateCredentialsEmailData struct {
	// Email of the credentials.
	//
	// Each email MUST be unique among all credentials. You may use the ExistsCredentialsEmailRepository to ensure
	// a given address is not already in use.
	//
	// Once an email is updated, the old value is freed and can be used by another set of credentials.
	Email string
	// Time at which the credentials were updated.
	Now time.Time
}

// UpdateCredentialsEmailRepository is the repository used to perform the
// UpdateCredentialsEmailRepository.UpdateCredentialsEmail action.
//
// You may create one using the NewUpdateCredentialsEmailRepository function.
type UpdateCredentialsEmailRepository struct{}

// UpdateCredentialsEmail updates the email of a set of credentials in the database.
//
// The new email MUST have been validated beforehand, and be unique among all credentials.
// The ExistsCredentialsEmailRepository can be used beforehand to ensure an email is available.
//
// Once the update completes, the old email is freed and can be used by another set of credentials.
func (repository *UpdateCredentialsEmailRepository) UpdateCredentialsEmail(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsEmailData,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{
		ID:        userID,
		Email:     data.Email,
		UpdatedAt: data.Now,
	}

	// Execute query.
	res, err := tx.NewUpdate().Model(entity).WherePK().Column("email", "updated_at").Returning("*").Exec(ctx)
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			return nil, NewErrUpdateCredentialsEmailRepository(errors.Join(err, ErrCredentialsAlreadyExists))
		}

		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("update credentials: %w", err))
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("get rows affected: %w", err))
	}

	if rowsAffected == 0 {
		return nil, NewErrUpdateCredentialsEmailRepository(ErrCredentialsNotFound)
	}

	return entity, nil
}

func NewUpdateCredentialsEmailRepository() *UpdateCredentialsEmailRepository {
	return &UpdateCredentialsEmailRepository{}
}
