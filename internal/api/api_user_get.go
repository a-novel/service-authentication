package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/api/codegen"
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
	span := sentry.StartSpan(ctx, "API.GetUser")
	defer span.Finish()

	span.SetData("request.userID", params.UserID)

	user, err := api.GetUserService.SelectUser(span.Context(), services.SelectUserRequest{
		ID: uuid.UUID(params.UserID),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound):
		span.SetData("service.err", err.Error())

		return &codegen.NotFoundError{Error: "user not found"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("get user: %w", err)
	}

	span.SetData("service.user", user)

	return &codegen.User{
		ID:        codegen.UserID(user.ID),
		Email:     codegen.Email(user.Email),
		Role:      api.CredentialsRoleFromModel(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}
