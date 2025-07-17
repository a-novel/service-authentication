package api

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type ListUsersService interface {
	ListUsers(ctx context.Context, request services.ListUsersRequest) ([]*models.User, error)
}

func (api *API) ListUsers(ctx context.Context, params apimodels.ListUsersParams) (apimodels.ListUsersRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.ListUsers")
	defer span.End()

	users, err := api.ListUsersService.ListUsers(ctx, services.ListUsersRequest{
		Limit:  params.Limit.Value,
		Offset: params.Offset.Value,
		Roles: lo.Map(params.Roles, func(item apimodels.CredentialsRole, _ int) models.CredentialsRole {
			return api.CredentialsRoleToModel(item)
		}),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list users: %w", err))
	}

	return otel.ReportSuccess(span, lo.ToPtr(apimodels.ListUsersOKApplicationJSON(
		lo.Map(users, func(item *models.User, _ int) apimodels.User {
			return apimodels.User{
				ID:        apimodels.UserID(item.ID),
				Email:     apimodels.Email(item.Email),
				Role:      api.CredentialsRoleFromModel(item.Role),
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
		}),
	))), nil
}
