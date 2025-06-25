package api

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"

	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type ListUsersService interface {
	ListUsers(ctx context.Context, request services.ListUsersRequest) ([]*models.User, error)
}

func (api *API) ListUsers(ctx context.Context, params codegen.ListUsersParams) (codegen.ListUsersRes, error) {
	span := sentry.StartSpan(ctx, "API.ListUsers")
	defer span.Finish()

	span.SetData("request.limit", params.Limit.Value)
	span.SetData("request.offset", params.Offset.Value)
	span.SetData("request.roles", params.Roles)

	users, err := api.ListUsersService.ListUsers(span.Context(), services.ListUsersRequest{
		Limit:  params.Limit.Value,
		Offset: params.Offset.Value,
		Roles: lo.Map(params.Roles, func(item codegen.CredentialsRole, _ int) models.CredentialsRole {
			return api.CredentialsRoleToModel(item)
		}),
	})
	if err != nil {
		span.SetData("service.err", err.Error())

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

	span.SetData("service.users.count", len(res))

	return &res, nil
}
