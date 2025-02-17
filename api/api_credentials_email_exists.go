package api

import (
	"errors"
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) EmailExists(ctx context.Context, params codegen.EmailExistsParams) (codegen.EmailExistsRes, error) {
	exists, err := api.EmailExistsService.EmailExists(ctx, services.EmailExistsRequest{
		Email: string(params.Email),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsNotFound), !exists:
		return &codegen.NotFoundError{Error: "email not found"}, nil
	case err != nil:
		return nil, fmt.Errorf("check email existence: %w", err)
	}

	return &codegen.EmailExistsNoContent{}, nil
}
