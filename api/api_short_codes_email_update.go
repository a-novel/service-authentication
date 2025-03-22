package api

import (
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/services"
	"github.com/a-novel/authentication/models"
)

type RequestEmailUpdateService interface {
	RequestEmailUpdate(ctx context.Context, request services.RequestEmailUpdateRequest) (*models.ShortCode, error)
}

func (api *API) RequestEmailUpdate(
	ctx context.Context, req *codegen.RequestEmailUpdateForm,
) (codegen.RequestEmailUpdateRes, error) {
	userID, err := RequireUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("require user ID: %w", err)
	}

	_, err = api.RequestEmailUpdateService.RequestEmailUpdate(ctx, services.RequestEmailUpdateRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
		ID:    userID,
	})
	if err != nil {
		return nil, fmt.Errorf("request email update: %w", err)
	}

	return &codegen.RequestEmailUpdateNoContent{}, nil
}
