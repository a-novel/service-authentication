package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
)

type UpdatePasswordService interface {
	UpdatePassword(ctx context.Context, request services.UpdatePasswordRequest) error
}

func (api *API) UpdatePassword(
	ctx context.Context, req *codegen.UpdatePasswordForm) (codegen.UpdatePasswordRes, error,
) {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("update password: %w", err)
	}

	err = api.UpdatePasswordService.UpdatePassword(ctx, services.UpdatePasswordRequest{
		Password:        string(req.GetPassword()),
		CurrentPassword: string(req.GetCurrentPassword()),
		UserID:          userID,
	})

	switch {
	case errors.Is(err, lib.ErrInvalidPassword):
		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		return nil, fmt.Errorf("update password: %w", err)
	}

	return &codegen.UpdatePasswordNoContent{}, nil
}
