package services

import (
	"context"
	"errors"

	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var ErrListUsersService = errors.New("ListUsersService.ListUsers")

func NewErrListUsersService(err error) error {
	return errors.Join(err, ErrListUsersService)
}

type ListUsersSource interface {
	ListUsers(ctx context.Context, data dao.ListUsersData) ([]*dao.CredentialsEntity, error)
}

type ListUsersRequest struct {
	Limit  int
	Offset int
	Roles  []models.CredentialsRole
}

type ListUsersService struct {
	source ListUsersSource
}

func (service *ListUsersService) ListUsers(
	ctx context.Context, request ListUsersRequest,
) ([]*models.User, error) {
	entities, err := service.source.ListUsers(ctx, dao.ListUsersData{
		Limit:  request.Limit,
		Offset: request.Offset,
		Roles:  request.Roles,
	})
	if err != nil {
		return nil, NewErrListUsersService(err)
	}

	return lo.Map(entities, func(item *dao.CredentialsEntity, _ int) *models.User {
		return &models.User{
			ID:        item.ID,
			Email:     item.Email,
			Role:      item.Role,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}), nil
}

func NewListUsersService(source ListUsersSource) *ListUsersService {
	return &ListUsersService{source: source}
}
