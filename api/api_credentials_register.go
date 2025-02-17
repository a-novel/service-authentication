package api

import (
	"errors"
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/services"
)

func (api *API) Register(ctx context.Context, req *codegen.RegisterForm) (codegen.RegisterRes, error) {
	accessToken, err := api.RegisterService.Register(ctx, services.RegisterRequest{
		Email:    string(req.Email),
		Password: string(req.Password),
	})

	switch {
	case errors.Is(err, dao.ErrCredentialsAlreadyExists):
		return &codegen.ConflictError{Error: "email already taken"}, nil
	case errors.Is(err, dao.ErrShortCodeNotFound), errors.Is(err, services.ErrInvalidShortCode):
		return &codegen.ForbiddenError{Error: "invalid short code"}, nil
	case err != nil:
		return nil, fmt.Errorf("register user: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
