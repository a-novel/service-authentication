package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type UpdateRoleService interface {
	UpdateRole(ctx context.Context, request services.UpdateRoleRequest) (*models.User, error)
}

func (api *API) UpdateRole(ctx context.Context, req *codegen.UpdateRoleForm) (codegen.UpdateRoleRes, error) {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user id: %w", err)
	}

	user, err := api.UpdateRoleService.UpdateRole(ctx, services.UpdateRoleRequest{
		TargetUserID:  uuid.UUID(req.GetUserID()),
		CurrentUserID: userID,
		Role:          api.CredentialsRoleToModel(req.GetRole()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		return &codegen.NotFoundError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUpdateToHigherRole), errors.Is(err, services.ErrMustDowngradeLowerRole):
		return &codegen.ForbiddenError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUnknownRole), errors.Is(err, services.ErrSelfRoleUpdate):
		return &codegen.UnprocessableEntityError{Error: err.Error()}, nil
	case err != nil:
		return nil, fmt.Errorf("update role: %w", err)
	}

	return &codegen.User{
		ID:        codegen.UserID(user.ID),
		Email:     codegen.Email(user.Email),
		Role:      api.CredentialsRoleFromModel(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
