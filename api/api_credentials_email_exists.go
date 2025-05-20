package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

type EmailExistsService interface {
	EmailExists(ctx context.Context, request services.EmailExistsRequest) (bool, error)
}

func (api *API) EmailExists(ctx context.Context, params codegen.EmailExistsParams) (codegen.EmailExistsRes, error) {
	exists, err := api.EmailExistsService.EmailExists(ctx, services.EmailExistsRequest{
		Email: string(params.Email),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound), !exists && err == nil:
		return &codegen.NotFoundError{Error: "email not found"}, nil
	case err != nil:
		return nil, fmt.Errorf("check email existence: %w", err)
	}

	return &codegen.EmailExistsNoContent{}, nil
}
