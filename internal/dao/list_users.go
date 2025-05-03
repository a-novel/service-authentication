package dao

import (
	"errors"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/service-authentication/models"
)

var ErrListUsersRepository = errors.New("ListUsersRepository.ListUsers")

func NewErrListUsersRepository(err error) error {
	return errors.Join(err, ErrListUsersRepository)
}

type ListUsersData struct {
	Limit  int
	Offset int
	Roles  []models.CredentialsRole
}

type ListUsersRepository struct{}

func (repository *ListUsersRepository) ListUsers(
	ctx context.Context, data ListUsersData,
) ([]*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, NewErrListUsersRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entities := make([]*CredentialsEntity, 0)

	query := tx.NewSelect().
		Model(&entities).
		Column("id", "email", "role", "created_at", "updated_at").
		Order("updated_at DESC").
		Limit(data.Limit).
		Offset(data.Offset)

	if len(data.Roles) > 0 {
		query = query.Where("role IN (?)", bun.In(data.Roles))
	}

	if err = query.Scan(ctx); err != nil {
		return nil, NewErrListUsersRepository(fmt.Errorf("list credentials: %w", err))
	}

	return entities, nil
}

func NewListUsersRepository() *ListUsersRepository {
	return &ListUsersRepository{}
}
