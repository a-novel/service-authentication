package services

import (
	"context"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

type ListUsersSource interface {
	ListUsers(ctx context.Context, data dao.ListUsersData) ([]*dao.CredentialsEntity, error)
}

type ListUsersRequest struct {
	Limit  int
	Offset int
	Roles  models.CredentialsRoles
}

type ListUsersService struct {
	source ListUsersSource
}

func NewListUsersService(source ListUsersSource) *ListUsersService {
	return &ListUsersService{source: source}
}

func (service *ListUsersService) ListUsers(
	ctx context.Context, request ListUsersRequest,
) ([]*models.User, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.ListUsers")
	defer span.End()

	span.SetAttributes(
		attribute.Int("request.limit", request.Limit),
		attribute.Int("request.offset", request.Offset),
		attribute.StringSlice("request.roles", request.Roles.Strings()),
	)

	entities, err := service.source.ListUsers(ctx, dao.ListUsersData{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.Int("response.count", len(entities)))

	return otel.ReportSuccess(span, lo.Map(entities, func(item *dao.CredentialsEntity, _ int) *models.User {
		return &models.User{
			ID:        item.ID,
			Email:     item.Email,
			Role:      item.Role,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	})), nil
}
