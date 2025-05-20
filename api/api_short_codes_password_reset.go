package api

import (
	"context"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type RequestPasswordResetService interface {
	RequestPasswordReset(ctx context.Context, request services.RequestPasswordResetRequest) (*models.ShortCode, error)
}

func (api *API) RequestPasswordReset(
	ctx context.Context, req *codegen.RequestPasswordResetForm,
) (codegen.RequestPasswordResetRes, error) {
	_, err := api.RequestPasswordResetService.RequestPasswordReset(ctx, services.RequestPasswordResetRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		return nil, fmt.Errorf("request password reset: %w", err)
	}

	return &codegen.RequestPasswordResetNoContent{}, nil
}
