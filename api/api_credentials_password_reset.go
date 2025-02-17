package api

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) ResetPassword(ctx context.Context, req *codegen.ResetPasswordForm) (codegen.ResetPasswordRes, error) {
	err := api.UpdatePasswordService.UpdatePassword(ctx, services.UpdatePasswordRequest{
		Password:  string(req.Password),
		ShortCode: string(req.ShortCode),
		UserID:    uuid.UUID(req.UserID),
	})

	switch {
	case errors.Is(err, lib.ErrInvalidPassword):
		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case err != nil:
		return nil, fmt.Errorf("reset password: %w", err)
	}

	return &codegen.ResetPasswordNoContent{}, nil
}
