package api

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type GetUserService interface {
	SelectUser(ctx context.Context, request services.SelectUserRequest) (*models.User, error)
}

func (api *API) GetUser(
	ctx context.Context, params codegen.GetUserParams,
) (codegen.GetUserRes, error) {
	user, err := api.GetUserService.SelectUser(ctx, services.SelectUserRequest{
		ID: uuid.UUID(params.UserID),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		return &codegen.NotFoundError{Error: "user not found"}, nil
	case err != nil:
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &codegen.User{
		ID:        codegen.UserID(user.ID),
		Email:     codegen.Email(user.Email),
		Role:      api.CredentialsRoleFromModel(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
