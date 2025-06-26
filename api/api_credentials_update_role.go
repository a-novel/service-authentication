package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
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
	span := sentry.StartSpan(ctx, "API.UpdateEmail")
	defer span.Finish()

	span.SetData("request.targetUserID", req.GetUserID())
	span.SetData("request.role", req.GetRole())

	userID, err := RequireUserID(span.Context())
	if err != nil {
		span.SetData("request.userID.err", err.Error())

		return nil, fmt.Errorf("get user id: %w", err)
	}

	span.SetData("request.userID", userID)

	user, err := api.UpdateRoleService.UpdateRole(span.Context(), services.UpdateRoleRequest{
		TargetUserID:  uuid.UUID(req.GetUserID()),
		CurrentUserID: userID,
		Role:          api.CredentialsRoleToModel(req.GetRole()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.SetData("service.err", err.Error())

		return &codegen.NotFoundError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUpdateToHigherRole), errors.Is(err, services.ErrMustDowngradeLowerRole):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUnknownRole), errors.Is(err, services.ErrSelfRoleUpdate):
		span.SetData("service.err", err.Error())

		return &codegen.UnprocessableEntityError{Error: err.Error()}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

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
