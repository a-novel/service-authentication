package dao

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/models"
)

//go:embed list_users.sql
var listUsersQuery string

type ListUsersData struct {
	Limit  int
	Offset int
	Roles  models.CredentialsRoles
}

type ListUsersRepository struct{}

func NewListUsersRepository() *ListUsersRepository {
	return &ListUsersRepository{}
}

func (repository *ListUsersRepository) ListUsers(
	ctx context.Context, data ListUsersData,
) ([]*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int("data.limit", data.Limit),
		attribute.Int("data.offset", data.Offset),
		attribute.StringSlice("data.roles", data.Roles.Strings()),
	)

	if len(data.Roles) == 0 {
		// Make sure roles are defined for the query.
		data.Roles = models.CredentialsRoles{}
	}

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entities := make([]*CredentialsEntity, 0, data.Limit)

	err = tx.NewRaw(
		listUsersQuery,
		bun.NullZero(data.Limit),
		data.Offset,
		bun.In(data.Roles),
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list credentials: %w", err))
	}

	return otel.ReportSuccess(span, entities), nil
}
