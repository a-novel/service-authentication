package services

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

type CredentialsExistRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsExistRequest) (bool, error)
}

type CredentialsExistRequest struct {
	Email string `validate:"required,email"`
}

type CredentialsExist struct {
	repository CredentialsExistRepository
}

func NewCredentialsExist(repository CredentialsExistRepository) *CredentialsExist {
	return &CredentialsExist{
		repository: repository,
	}
}

func (service *CredentialsExist) Exec(ctx context.Context, request *CredentialsExistRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsExist")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return false, errors.Join(err, ErrInvalidRequest)
	}

	exists, err := service.repository.Exec(ctx, &dao.CredentialsExistRequest{
		Email: request.Email,
	})
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	span.SetAttributes(attribute.Bool("exists", exists))

	return otel.ReportSuccess(span, exists), nil
}
