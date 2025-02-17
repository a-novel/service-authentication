package dao

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

// UpdateCredentialsPasswordData is the input used to perform the
// UpdateCredentialsPasswordRepository.UpdateCredentialsPassword action.
type UpdateCredentialsPasswordData struct {
	// The user's password, encrypted using scrypt. This will be used when creating a session.
	//
	// This value is sensitive, and MUST always be transmitted using a secure channel / encryption. It MUST NEVER be
	// stored clear in the database.
	Password string
	// Time at which the credentials were updated.
	Now time.Time
}

// UpdateCredentialsPasswordRepository is the repository used to perform the
// UpdateCredentialsPasswordRepository.UpdateCredentialsPassword action.
//
// You may create one using the NewUpdateCredentialsPasswordRepository function.
type UpdateCredentialsPasswordRepository struct{}

// UpdateCredentialsPassword updates the password of a set of credentials in the database.
//
// This method MUST be called after extra checks on the user identity have been performed. This is a sensitive
// operation.
func (repository *UpdateCredentialsPasswordRepository) UpdateCredentialsPassword(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsPasswordData,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"(UpdateCredentialsPasswordRepository.UpdateCredentialsPassword) get postgres client: %w", err,
		)
	}

	entity := &CredentialsEntity{
		ID:        userID,
		Password:  data.Password,
		UpdatedAt: data.Now,
	}

	// Execute query.
	res, err := tx.NewUpdate().Model(entity).WherePK().Column("password", "updated_at").Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"(UpdateCredentialsPasswordRepository.UpdateCredentialsPassword) update credentials: %w", err,
		)
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"(UpdateCredentialsPasswordRepository.UpdateCredentialsPassword) get rows affected: %w", err,
		)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf(
			"(UpdateCredentialsPasswordRepository.UpdateCredentialsPassword) %w", ErrCredentialsNotFound,
		)
	}

	return entity, nil
}

func NewUpdateCredentialsPasswordRepository() *UpdateCredentialsPasswordRepository {
	return &UpdateCredentialsPasswordRepository{}
}
