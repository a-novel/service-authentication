package api

import (
	"context"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type RequestRegisterService interface {
	RequestRegister(ctx context.Context, request services.RequestRegisterRequest) (*models.ShortCode, error)
}

func (api *API) RequestRegistration(
	ctx context.Context, req *codegen.RequestRegistrationForm,
) (codegen.RequestRegistrationRes, error) {
	_, err := api.RequestRegisterService.RequestRegister(ctx, services.RequestRegisterRequest{
		Email: string(req.GetEmail()),
		Lang:  models.Lang(req.GetLang().Value),
	})
	if err != nil {
		return nil, fmt.Errorf("request register: %w", err)
	}

	return &codegen.RequestRegistrationNoContent{}, nil
}
