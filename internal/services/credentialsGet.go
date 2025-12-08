package services

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsGetRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsGetRequest struct {
	ID uuid.UUID
}

type CredentialsGet struct {
	repository CredentialsGetRepository
}

func NewCredentialsGet(repository CredentialsGetRepository) *CredentialsGet {
	return &CredentialsGet{
		repository: repository,
	}
}

func (service *CredentialsGet) Exec(
	ctx context.Context, request *CredentialsGetRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsGet")
	defer span.End()

	span.SetAttributes(attribute.String("request.userID", request.ID.String()))

	entity, err := service.repository.Exec(ctx, &dao.CredentialsSelectRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("dao.entity.id", entity.ID.String()),
		attribute.String("dao.entity.email", entity.Email),
		attribute.String("dao.entity.role", entity.Role),
	)

	return otel.ReportSuccess(span, &Credentials{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}), nil
}
