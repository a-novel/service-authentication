package api

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type ListUsersService interface {
	ListUsers(ctx context.Context, request services.ListUsersRequest) ([]*models.User, error)
}

func (api *API) ListUsers(ctx context.Context, params codegen.ListUsersParams) (codegen.ListUsersRes, error) {
	users, err := api.ListUsersService.ListUsers(ctx, services.ListUsersRequest{
		Limit:  params.Limit.Value,
		Offset: params.Offset.Value,
		Roles: lo.Map(params.Roles, func(item codegen.CredentialsRole, _ int) models.CredentialsRole {
			return api.CredentialsRoleToModel(item)
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	res := codegen.ListUsersOKApplicationJSON(
		lo.Map(users, func(item *models.User, _ int) codegen.User {
			return codegen.User{
				ID:        codegen.UserID(item.ID),
				Email:     codegen.Email(item.Email),
				Role:      api.CredentialsRoleFromModel(item.Role),
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
		}),
	)

	return &res, nil
}
