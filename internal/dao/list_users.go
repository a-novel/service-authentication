package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/uptrace/bun"

	"github.com/a-novel/service-authentication/internal/lib"
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

func NewListUsersRepository() *ListUsersRepository {
	return &ListUsersRepository{}
}

func (repository *ListUsersRepository) ListUsers(
	ctx context.Context, data ListUsersData,
) ([]*CredentialsEntity, error) {
	span := sentry.StartSpan(ctx, "ListUsersRepository.ListUsers")
	defer span.Finish()

	span.SetData("limit", data.Limit)
	span.SetData("offset", data.Offset)
	span.SetData("roles", data.Roles)

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

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

	err = query.Scan(span.Context())
	if err != nil {
		span.SetData("scan.error", err.Error())

		return nil, NewErrListUsersRepository(fmt.Errorf("list credentials: %w", err))
	}

	return entities, nil
}
