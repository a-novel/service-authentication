package core

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsGetDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

type CredentialsGetRequest struct {
	ID uuid.UUID
}

// CredentialsGet retrieves a single account by its ID.
type CredentialsGet struct {
	dao CredentialsGetDao
}

func NewCredentialsGet(dao CredentialsGetDao) *CredentialsGet {
	return &CredentialsGet{
		dao: dao,
	}
}

func (service *CredentialsGet) Exec(
	ctx context.Context, request *CredentialsGetRequest,
) (*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsGet")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.ID.String()))

	entity, err := service.dao.Exec(ctx, &dao.CredentialsSelectRequest{ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	span.SetAttributes(
		attribute.String("user.email", entity.Email),
		attribute.String("user.role", entity.Role),
	)

	return &Credentials{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}, nil
}
