package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsListRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsListRequest) ([]*dao.Credentials, error)
}

type CredentialsListRequest struct {
	Limit  int      `validate:"required,max=100"`
	Offset int      `validate:"min=0"`
	Roles  []string `validate:"min=0,max=10,dive,role"`
}

type CredentialsList struct {
	repository CredentialsListRepository
}

func NewCredentialsList(repository CredentialsListRepository) *CredentialsList {
	return &CredentialsList{
		repository: repository,
	}
}

func (service *CredentialsList) Exec(
	ctx context.Context, request *CredentialsListRequest,
) ([]*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsList")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entities, err := service.repository.Exec(ctx, &dao.CredentialsListRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list credentials: %w", err))
	}

	span.SetAttributes(attribute.Int("response.count", len(entities)))

	return otel.ReportSuccess(span, lo.Map(entities, func(item *dao.Credentials, _ int) *Credentials {
		return &Credentials{
			ID:        item.ID,
			Email:     item.Email,
			Role:      item.Role,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	})), nil
}
