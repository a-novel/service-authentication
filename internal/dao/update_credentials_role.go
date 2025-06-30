package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrUpdateCredentialsRoleRepository = errors.New("UpdateCredentialsRoleRepository.UpdateCredentialsRole")

func NewErrUpdateCredentialsRoleRepository(err error) error {
	return errors.Join(err, ErrUpdateCredentialsRoleRepository)
}

type UpdateCredentialsRoleData struct {
	Role models.CredentialsRole
	Now  time.Time
}

type UpdateCredentialsRoleRepository struct{}

func NewUpdateCredentialsRoleRepository() *UpdateCredentialsRoleRepository {
	return &UpdateCredentialsRoleRepository{}
}

func (repository *UpdateCredentialsRoleRepository) UpdateCredentialsRole(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsRoleData,
) (*CredentialsEntity, error) {
	span := sentry.StartSpan(ctx, "UpdateCredentialsRoleRepository.UpdateCredentialsRole")
	defer span.Finish()

	span.SetData("credentials.id", userID.String())
	span.SetData("credentials.role", data.Role)
	span.SetData("credentials.now", data.Now.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrUpdateCredentialsRoleRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{
		ID:        userID,
		Role:      data.Role,
		UpdatedAt: data.Now,
	}

	// Execute query.
	res, err := tx.NewUpdate().
		Model(entity).
		WherePK().
		Column("role", "updated_at").
		Returning("*").
		Exec(span.Context())
	if err != nil {
		span.SetData("update.error", err.Error())

		return nil, NewErrUpdateCredentialsRoleRepository(fmt.Errorf("update credentials: %w", err))
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.SetData("rowsAffected.error", err.Error())

		return nil, NewErrUpdateCredentialsRoleRepository(fmt.Errorf("get rows affected: %w", err))
	}

	span.SetData("rowsAffected", rowsAffected)

	if rowsAffected == 0 {
		span.SetData("error", "credentials not found")

		return nil, NewErrUpdateCredentialsRoleRepository(ErrCredentialsNotFound)
	}

	return entity, nil
}
