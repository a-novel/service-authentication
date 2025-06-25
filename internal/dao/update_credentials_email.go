package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/getsentry/sentry-go"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
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
	span := sentry.StartSpan(ctx, "UpdateCredentialsEmailRepository.UpdateCredentialsEmail")
	defer span.Finish()

	span.SetData("credentials.id", userID.String())
	span.SetData("credentials.email", data.Email)
	span.SetData("credentials.now", data.Now.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{
		ID:        userID,
		Email:     data.Email,
		UpdatedAt: data.Now,
	}

	// Execute query.
	res, err := tx.NewUpdate().Model(entity).WherePK().Column("email", "updated_at").Returning("*").Exec(span.Context())
	if err != nil {
		span.SetData("update.error", err.Error())

		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			return nil, NewErrUpdateCredentialsEmailRepository(errors.Join(err, ErrCredentialsAlreadyExists))
		}

		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("update credentials: %w", err))
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.SetData("rowsAffected.error", err.Error())

		return nil, NewErrUpdateCredentialsEmailRepository(fmt.Errorf("get rows affected: %w", err))
	}

	span.SetData("rowsAffected", rowsAffected)

	if rowsAffected == 0 {
		span.SetData("error", "credentials not found")

		return nil, NewErrUpdateCredentialsEmailRepository(ErrCredentialsNotFound)
	}

	return entity, nil
}

func NewUpdateCredentialsEmailRepository() *UpdateCredentialsEmailRepository {
	return &UpdateCredentialsEmailRepository{}
}
