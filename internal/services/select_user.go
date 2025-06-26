package services

import (
	"context"
	"errors"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var ErrSelectUserService = errors.New("SelectUserService.SelectUser")

func NewErrSelectUserService(err error) error {
	return errors.Join(err, ErrSelectUserService)
}

type SelectUserSource interface {
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
}

type SelectUserRequest struct {
	ID uuid.UUID
}

type SelectUserService struct {
	source SelectUserSource
}

func (service *SelectUserService) SelectUser(
	ctx context.Context, request SelectUserRequest,
) (*models.User, error) {
	span := sentry.StartSpan(ctx, "SelectUserService.SelectUser")
	defer span.Finish()

	span.SetData("request.userID", request.ID.String())

	entity, err := service.source.SelectCredentials(span.Context(), request.ID)
	if err != nil {
		span.SetData("dao.error", err.Error())

		return nil, NewErrSelectUserService(err)
	}

	span.SetData("dao.entity.email", entity.Email)
	span.SetData("dao.entity.role", entity.Role)

	return &models.User{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}, nil
}

func NewSelectUserService(source SelectUserSource) *SelectUserService {
	return &SelectUserService{source: source}
}
