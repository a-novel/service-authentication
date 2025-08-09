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
)

type GetUserService interface {
	SelectUser(ctx context.Context, request services.SelectUserRequest) (*models.User, error)
}

func (api *API) GetUser(
	ctx context.Context, params apimodels.GetUserParams,
) (apimodels.GetUserRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.GetUser")
	defer span.End()

	user, err := api.GetUserService.SelectUser(ctx, services.SelectUserRequest{
		ID: uuid.UUID(params.UserID),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		_ = otel.ReportError(span, err)

		return &apimodels.NotFoundError{Error: "user not found"}, nil
	case err != nil:
		_ = otel.ReportError(span, err)

		return nil, fmt.Errorf("get user: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.User{
		ID:        apimodels.UserID(user.ID),
		Email:     apimodels.Email(user.Email),
		Role:      api.CredentialsRoleFromModel(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}), nil
}
