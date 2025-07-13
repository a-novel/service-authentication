package services

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

type SelectUserSource interface {
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
}

type SelectUserRequest struct {
	ID uuid.UUID
}

type SelectUserService struct {
	source SelectUserSource
}

func NewSelectUserService(source SelectUserSource) *SelectUserService {
	return &SelectUserService{source: source}
}

func (service *SelectUserService) SelectUser(
	ctx context.Context, request SelectUserRequest,
) (*models.User, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.SelectUser")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.ID.String()))

	entity, err := service.source.SelectCredentials(ctx, request.ID)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("dao.entity.id", entity.ID.String()),
		attribute.String("dao.entity.email", entity.Email),
		attribute.String("dao.entity.role", entity.Role.String()),
	)

	return otel.ReportSuccess(span, &models.User{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}), nil
}
