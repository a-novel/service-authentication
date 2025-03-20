package dao

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/models"
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

func (repository *UpdateCredentialsRoleRepository) UpdateCredentialsRole(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsRoleData,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
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
		Exec(ctx)
	if err != nil {
		return nil, NewErrUpdateCredentialsRoleRepository(fmt.Errorf("update credentials: %w", err))
	}

	// Make sure the credentials were updated.
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, NewErrUpdateCredentialsRoleRepository(fmt.Errorf("get rows affected: %w", err))
	}

	if rowsAffected == 0 {
		return nil, NewErrUpdateCredentialsRoleRepository(ErrCredentialsNotFound)
	}

	return entity, nil
}

func NewUpdateCredentialsRoleRepository() *UpdateCredentialsRoleRepository {
	return &UpdateCredentialsRoleRepository{}
}
