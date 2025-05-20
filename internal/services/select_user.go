package services

import (
	"context"
	"errors"

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
	entity, err := service.source.SelectCredentials(ctx, request.ID)
	if err != nil {
		return nil, NewErrSelectUserService(err)
	}

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
