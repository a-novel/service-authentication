package api

import (
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) RequestPasswordUpdate(
	ctx context.Context, req *codegen.RequestPasswordUpdateForm,
) (codegen.RequestPasswordUpdateRes, error) {
	_, err := api.RequestPasswordResetService.RequestPasswordReset(ctx, services.RequestPasswordResetRequest{
		Email: string(req.Email),
	})
	if err != nil {
		return nil, fmt.Errorf("request password reset: %w", err)
	}

	return &codegen.RequestPasswordUpdateNoContent{}, nil
}
