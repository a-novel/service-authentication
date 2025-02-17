package api

import (
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) RequestRegistration(
	ctx context.Context, req *codegen.RequestRegistrationForm,
) (codegen.RequestRegistrationRes, error) {
	_, err := api.RequestRegisterService.RequestRegister(ctx, services.RequestRegisterRequest{
		Email: string(req.Email),
	})
	if err != nil {
		return nil, fmt.Errorf("request register: %w", err)
	}

	return &codegen.RequestRegistrationNoContent{}, nil
}
