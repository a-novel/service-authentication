package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

type UpdateRoleService interface {
	UpdateRole(ctx context.Context, request services.UpdateRoleRequest) (*models.User, error)
}

func (api *API) UpdateRole(ctx context.Context, req *apimodels.UpdateRoleForm) (apimodels.UpdateRoleRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.UpdateRole")
	defer span.End()

	userID, err := pkg.RequireUserID(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get user id: %w", err))
	}

	user, err := api.UpdateRoleService.UpdateRole(ctx, services.UpdateRoleRequest{
		TargetUserID:  uuid.UUID(req.GetUserID()),
		CurrentUserID: userID,
		Role:          api.CredentialsRoleToModel(req.GetRole()),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		_ = otel.ReportError(span, err)

		return &apimodels.NotFoundError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUpdateToHigherRole), errors.Is(err, services.ErrMustDowngradeLowerRole):
		_ = otel.ReportError(span, err)

		return &apimodels.ForbiddenError{Error: err.Error()}, nil
	case errors.Is(err, services.ErrUnknownRole), errors.Is(err, services.ErrSelfRoleUpdate):
		_ = otel.ReportError(span, err)

		return &apimodels.UnprocessableEntityError{Error: err.Error()}, nil
	case err != nil:
		_ = otel.ReportError(span, err)

		return nil, fmt.Errorf("update role: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.User{
		ID:        apimodels.UserID(user.ID),
		Email:     apimodels.Email(user.Email),
		Role:      api.CredentialsRoleFromModel(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}), nil
}
