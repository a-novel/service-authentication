package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/api/codegen"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/pkg"
)

type UpdatePasswordService interface {
	UpdatePassword(ctx context.Context, request services.UpdatePasswordRequest) error
}

func (api *API) UpdatePassword(
	ctx context.Context, req *codegen.UpdatePasswordForm) (codegen.UpdatePasswordRes, error,
) {
	span := sentry.StartSpan(ctx, "API.UpdatePassword")
	defer span.Finish()

	userID, err := pkg.RequireUserID(span.Context())
	if err != nil {
		span.SetData("request.userID.err", err.Error())

		return nil, fmt.Errorf("get user id: %w", err)
	}

	span.SetData("request.userID", userID)

	err = api.UpdatePasswordService.UpdatePassword(span.Context(), services.UpdatePasswordRequest{
		Password:        string(req.GetPassword()),
		CurrentPassword: string(req.GetCurrentPassword()),
		UserID:          userID,
	})

	switch {
	case errors.Is(err, lib.ErrInvalidPassword):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("update password: %w", err)
	}

	return &codegen.UpdatePasswordNoContent{}, nil
}
