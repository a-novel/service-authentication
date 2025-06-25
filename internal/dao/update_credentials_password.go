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

var ErrUpdateCredentialsPasswordRepository = errors.New("UpdateCredentialsPasswordRepository.UpdateCredentialsPassword")

func NewErrUpdateCredentialsPasswordRepository(err error) error {
	return errors.Join(err, ErrUpdateCredentialsPasswordRepository)
}

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
	span := sentry.StartSpan(ctx, "UpdateCredentialsPasswordRepository.UpdateCredentialsPassword")
	defer span.Finish()

	span.SetData("credentials.id", userID.String())
	span.SetData("credentials.now", data.Now.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrUpdateCredentialsPasswordRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{
		ID:        userID,
		Password:  data.Password,
		UpdatedAt: data.Now,
	}

	// Execute query.
	res, err := tx.
		NewUpdate().
		Model(entity).
		WherePK().
		Column("password", "updated_at").
		Returning("*").
		Exec(span.Context())
	if err != nil {
		span.SetData("update.error", err.Error())

		return nil, NewErrUpdateCredentialsPasswordRepository(fmt.Errorf("update credentials: %w", err))
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.SetData("rowsAffected.error", err.Error())

		return nil, NewErrUpdateCredentialsPasswordRepository(fmt.Errorf("get rows affected: %w", err))
	}

	span.SetData("rowsAffected", rowsAffected)

	if rowsAffected == 0 {
		span.SetData("error", "credentials not found")

		return nil, NewErrUpdateCredentialsPasswordRepository(ErrCredentialsNotFound)
	}

	return entity, nil
}

func NewUpdateCredentialsPasswordRepository() *UpdateCredentialsPasswordRepository {
	return &UpdateCredentialsPasswordRepository{}
}
